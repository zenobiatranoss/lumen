package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"unicode/utf8"

	"lumen/internal/fsutil"
	"lumen/internal/media"
)

var extraTypes = map[string]string{
	"webp": "image/webp", "avif": "image/avif", "heic": "image/heic", "heif": "image/heif",
	"jxl": "image/jxl", "tif": "image/tiff", "tiff": "image/tiff", "bmp": "image/bmp",
	"svg": "image/svg+xml", "ico": "image/x-icon", "apng": "image/apng",
	"mkv": "video/x-matroska", "mka": "audio/x-matroska", "webm": "video/webm",
	"mp4": "video/mp4", "m4v": "video/mp4", "mov": "video/quicktime", "ogv": "video/ogg",
	"opus": "audio/ogg", "oga": "audio/ogg", "flac": "audio/flac", "m4a": "audio/mp4",
	"aac": "audio/aac", "mp3": "audio/mpeg", "wav": "audio/wav",
	"pdf": "application/pdf",
}

func contentType(e fsutil.Entry) string {
	if t, ok := extraTypes[e.Ext]; ok {
		return t
	}
	switch e.Kind {
	case fsutil.KindText, fsutil.KindCode:
		return "text/plain; charset=utf-8"
	}
	if t := mime.TypeByExtension("." + e.Ext); t != "" && e.Ext != "" {
		return t
	}
	return "application/octet-stream"
}

func (s *Server) file(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := fsutil.Stat(path)
	if err != nil {
		fail(w, err)
		return
	}
	if entry.Kind == fsutil.KindFolder {
		writeError(w, http.StatusBadRequest, "that path is a folder")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		fail(w, err)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		fail(w, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", contentType(entry))
	h.Set("Cache-Control", "private, max-age=300")
	if entry.Ext == "svg" {
		h.Set("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:")
	}
	if r.URL.Query().Get("dl") == "1" {
		h.Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(entry.Name))
	}
	http.ServeContent(w, r, entry.Name, info.ModTime(), f)
}

func (s *Server) thumb(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := fsutil.Stat(path)
	if err != nil {
		fail(w, err)
		return
	}
	if entry.Kind == fsutil.KindFolder {
		writeError(w, http.StatusBadRequest, "folders have no thumbnail")
		return
	}
	width, _ := strconv.Atoi(r.URL.Query().Get("w"))
	if width <= 0 {
		width = 320
	}
	cached, err := s.opts.Thumbs.Get(r.Context(), entry, width)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrUnsupported):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			writeError(w, http.StatusUnprocessableEntity, "could not create a preview")
		}
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=604800")
	http.ServeFile(w, r, cached)
}

func (s *Server) text(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := 256 << 10
	if v, err := strconv.Atoi(r.URL.Query().Get("max")); err == nil && v > 0 && v <= 2<<20 {
		limit = v
	}
	f, err := os.Open(path)
	if err != nil {
		fail(w, err)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		fail(w, err)
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "that path is a folder")
		return
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)))
	if err != nil {
		fail(w, err)
		return
	}
	probe := data
	if len(probe) > 8192 {
		probe = probe[:8192]
	}
	if bytes.IndexByte(probe, 0) >= 0 {
		writeError(w, http.StatusUnsupportedMediaType, "this file is not text")
		return
	}
	if !utf8.Valid(data) {
		data = bytes.ToValidUTF8(data, []byte("�"))
	}
	h := w.Header()
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Mtime", strconv.FormatInt(info.ModTime().UnixMilli(), 10))
	if info.Size() > int64(limit) {
		h.Set("X-Truncated", "1")
	}
	w.Write(data)
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tools := s.opts.Thumbs.Tools()
	if tools.FFmpeg == "" {
		writeError(w, http.StatusNotImplemented, "ffmpeg is not installed, so this video cannot be converted for playback")
		return
	}
	if !isFile(path) {
		writeError(w, http.StatusNotFound, "that file does not exist")
		return
	}
	start, _ := strconv.ParseFloat(r.URL.Query().Get("t"), 64)
	if start < 0 {
		start = 0
	}
	probe, _ := tools.Probe(r.Context(), path)
	h := w.Header()
	h.Set("Content-Type", "video/mp4")
	h.Set("Cache-Control", "no-store")
	h.Set("Accept-Ranges", "none")
	if err := tools.Stream(r.Context(), w, path, start, probe); err != nil && r.Context().Err() == nil {
		fmt.Fprintln(os.Stderr, "stream:", err)
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func (s *Server) cacheStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.opts.Thumbs.Stats())
}

func (s *Server) cacheClear(w http.ResponseWriter, r *http.Request) {
	if err := s.opts.Thumbs.Clear(); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.opts.Thumbs.Stats())
}
