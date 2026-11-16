package media

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"lumen/internal/fsutil"
)

var ErrUnsupported = errors.New("no preview is available for this file")

var widthBuckets = []int{160, 240, 320, 480, 640, 960, 1440, 2048, 2560}

func Bucket(width int) int {
	for _, b := range widthBuckets {
		if width <= b {
			return b
		}
	}
	return widthBuckets[len(widthBuckets)-1]
}

type flight struct {
	done chan struct{}
	path string
	err  error
}

type Thumbs struct {
	dir     string
	tools   Tools
	sem     chan struct{}
	mu      sync.Mutex
	pending map[string]*flight
}

func NewThumbs(dir string, tools Tools) *Thumbs {
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 6 {
		workers = 6
	}
	return &Thumbs{
		dir:     dir,
		tools:   tools,
		sem:     make(chan struct{}, workers),
		pending: make(map[string]*flight),
	}
}

func (t *Thumbs) Tools() Tools {
	return t.tools
}

func cacheKey(e fsutil.Entry, width int) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("v3|%s|%d|%d|%d", e.Path, e.ModTime, e.Size, width)))
	return hex.EncodeToString(sum[:])
}

func (t *Thumbs) cachedPath(key string) (string, bool) {
	base := filepath.Join(t.dir, key[:2], key[2:])
	for _, ext := range []string{".jpg", ".png"} {
		if _, err := os.Stat(base + ext); err == nil {
			return base + ext, true
		}
	}
	return "", false
}

func (t *Thumbs) Get(ctx context.Context, e fsutil.Entry, width int) (string, error) {
	width = Bucket(width)
	key := cacheKey(e, width)
	if path, ok := t.cachedPath(key); ok {
		return path, nil
	}
	t.mu.Lock()
	if f, ok := t.pending[key]; ok {
		t.mu.Unlock()
		return t.wait(ctx, f)
	}
	f := &flight{done: make(chan struct{})}
	t.pending[key] = f
	t.mu.Unlock()
	go t.run(ctx, key, e, width, f)
	return t.wait(ctx, f)
}

func (t *Thumbs) wait(ctx context.Context, f *flight) (string, error) {
	select {
	case <-f.done:
		return f.path, f.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (t *Thumbs) run(ctx context.Context, key string, e fsutil.Entry, width int, f *flight) {
	defer func() {
		t.mu.Lock()
		delete(t.pending, key)
		t.mu.Unlock()
		close(f.done)
	}()
	select {
	case t.sem <- struct{}{}:
	case <-ctx.Done():
		f.err = ctx.Err()
		return
	}
	defer func() { <-t.sem }()
	data, ext, err := t.render(e, width)
	if err != nil {
		f.err = err
		return
	}
	dir := filepath.Join(t.dir, key[:2])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.err = err
		return
	}
	out := filepath.Join(dir, key[2:]+"."+ext)
	tmp := out + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		f.err = err
		return
	}
	if err := os.Rename(tmp, out); err != nil {
		f.err = err
		return
	}
	f.path = out
}

func goDecodable(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "jpe", "jfif", "png", "gif":
		return true
	}
	return false
}

func (t *Thumbs) render(e fsutil.Entry, width int) ([]byte, string, error) {
	switch {
	case goDecodable(e.Ext):
		data, ext, err := renderGo(e.Path, e.Ext, width)
		if err == nil {
			return data, ext, nil
		}
		if data, ext, ferr := t.renderExternal(e, width); ferr == nil {
			return data, ext, nil
		}
		return nil, "", err
	case e.Kind == fsutil.KindVideo || e.Kind == fsutil.KindAudio:
		return t.renderFFmpeg(e, width)
	case e.Ext == "pdf":
		return t.renderPDF(e, width)
	case e.Kind == fsutil.KindImage && e.Ext != "svg":
		return t.renderExternal(e, width)
	}
	return nil, "", ErrUnsupported
}

func encode(img *image.RGBA, width int) ([]byte, string, error) {
	var buf bytes.Buffer
	if !img.Opaque() {
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := enc.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "png", nil
	}
	quality := 82
	if width > 640 {
		quality = 88
	}
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "jpg", nil
}

func renderGo(path, ext string, width int) ([]byte, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, "", err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, "", ErrUnsupported
	}
	o := 1
	if ext != "png" && ext != "gif" {
		o = Orientation(path)
	}
	shownWidth := w
	if o >= 5 {
		shownWidth = h
	}
	scale := math.Min(1, float64(width)/float64(shownWidth))
	tw := int(math.Round(float64(w) * scale))
	th := int(math.Round(float64(h) * scale))
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	out := orient(resample(img, tw, th), o)
	return encode(out, width)
}

func (t *Thumbs) renderExternal(e fsutil.Entry, width int) ([]byte, string, error) {
	if data, ext, err := t.renderFFmpeg(e, width); err == nil {
		return data, ext, nil
	}
	return t.renderConvert(e, width)
}

func (t *Thumbs) seekFor(ctx context.Context, path string) float64 {
	p, err := t.tools.Probe(ctx, path)
	if err != nil || p.Duration < 4 {
		return 0
	}
	return math.Min(p.Duration*0.1, 60)
}

func (t *Thumbs) renderFFmpeg(e fsutil.Entry, width int) ([]byte, string, error) {
	if t.tools.FFmpeg == "" {
		return nil, "", ErrUnsupported
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	seek := 0.0
	if e.Kind == fsutil.KindVideo && t.tools.FFprobe != "" {
		seek = t.seekFor(ctx, e.Path)
	}
	attempt := func(at float64) ([]byte, error) {
		args := []string{"-v", "error", "-nostdin"}
		if at > 0 {
			args = append(args, "-ss", fmt.Sprintf("%.2f", at))
		}
		args = append(args,
			"-i", e.Path,
			"-map", "0:v:0",
			"-frames:v", "1",
			"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", width),
			"-q:v", "3",
			"-f", "mjpeg",
			"pipe:1",
		)
		out, err := exec.CommandContext(ctx, t.tools.FFmpeg, args...).Output()
		if err == nil && len(out) == 0 {
			err = ErrUnsupported
		}
		return out, err
	}
	data, err := attempt(seek)
	if err != nil && seek > 0 {
		data, err = attempt(0)
	}
	if err != nil {
		return nil, "", ErrUnsupported
	}
	return data, "jpg", nil
}

func (t *Thumbs) renderConvert(e fsutil.Entry, width int) ([]byte, string, error) {
	if t.tools.Convert == "" {
		return nil, "", ErrUnsupported
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{
		e.Path + "[0]",
		"-auto-orient",
		"-background", "#262624",
		"-alpha", "remove",
		"-alpha", "off",
		"-thumbnail", fmt.Sprintf("%dx>", width),
		"-quality", "85",
		"jpg:-",
	}
	out, err := exec.CommandContext(ctx, t.tools.Convert, args...).Output()
	if err != nil || len(out) == 0 {
		return nil, "", ErrUnsupported
	}
	return out, "jpg", nil
}

func (t *Thumbs) renderPDF(e fsutil.Entry, width int) ([]byte, string, error) {
	if t.tools.PDFToPPM == "" {
		return nil, "", ErrUnsupported
	}
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return nil, "", err
	}
	work, err := os.MkdirTemp(t.dir, "pdf-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(work)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	prefix := filepath.Join(work, "page")
	cmd := exec.CommandContext(ctx, t.tools.PDFToPPM,
		"-jpeg", "-f", "1", "-l", "1", "-singlefile",
		"-scale-to-x", fmt.Sprint(width), "-scale-to-y", "-1",
		e.Path, prefix)
	if err := cmd.Run(); err != nil {
		return nil, "", ErrUnsupported
	}
	data, err := os.ReadFile(prefix + ".jpg")
	if err != nil || len(data) == 0 {
		return nil, "", ErrUnsupported
	}
	return data, "jpg", nil
}

type CacheStats struct {
	Bytes int64 `json:"bytes"`
	Files int   `json:"files"`
}

func (t *Thumbs) Stats() CacheStats {
	var st CacheStats
	filepath.WalkDir(t.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			st.Bytes += info.Size()
			st.Files++
		}
		return nil
	})
	return st
}

func (t *Thumbs) Clear() error {
	return os.RemoveAll(t.dir)
}
