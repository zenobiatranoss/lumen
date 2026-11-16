package media

import (
	"encoding/binary"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"sync"
	"time"

	"lumen/internal/fsutil"
)

var dimExts = map[string]bool{
	"jpg": true, "jpeg": true, "jpe": true, "jfif": true,
	"png": true, "gif": true, "webp": true, "bmp": true,
}

func webpSize(head []byte) (int, int) {
	if len(head) < 30 || string(head[:4]) != "RIFF" || string(head[8:12]) != "WEBP" {
		return 0, 0
	}
	switch string(head[12:16]) {
	case "VP8 ":
		if head[23] != 0x9d || head[24] != 0x01 || head[25] != 0x2a {
			return 0, 0
		}
		w := int(binary.LittleEndian.Uint16(head[26:])) & 0x3fff
		h := int(binary.LittleEndian.Uint16(head[28:])) & 0x3fff
		return w, h
	case "VP8L":
		if head[20] != 0x2f {
			return 0, 0
		}
		bits := binary.LittleEndian.Uint32(head[21:])
		return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1
	case "VP8X":
		w := int(head[24]) | int(head[25])<<8 | int(head[26])<<16
		h := int(head[27]) | int(head[28])<<8 | int(head[29])<<16
		return w + 1, h + 1
	}
	return 0, 0
}

func bmpSize(head []byte) (int, int) {
	if len(head) < 26 || head[0] != 'B' || head[1] != 'M' {
		return 0, 0
	}
	w := int(int32(binary.LittleEndian.Uint32(head[18:])))
	h := int(int32(binary.LittleEndian.Uint32(head[22:])))
	if h < 0 {
		h = -h
	}
	return w, h
}

func ImageSize(path, ext string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	switch ext {
	case "webp", "bmp":
		head := make([]byte, 32)
		n, _ := f.Read(head)
		if ext == "webp" {
			return webpSize(head[:n])
		}
		return bmpSize(head[:n])
	}
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	w, h := cfg.Width, cfg.Height
	if ext == "jpg" || ext == "jpeg" || ext == "jpe" || ext == "jfif" {
		if Orientation(path) >= 5 {
			w, h = h, w
		}
	}
	return w, h
}

type dimValue struct{ w, h int }

type DimCache struct {
	mu sync.Mutex
	m  map[string]dimValue
}

func NewDimCache() *DimCache {
	return &DimCache{m: make(map[string]dimValue)}
}

func dimKey(e fsutil.Entry) string {
	return e.Path + "|" + strconv.FormatInt(e.ModTime, 10) + "|" + strconv.FormatInt(e.Size, 10)
}

func (c *DimCache) lookup(key string) (dimValue, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *DimCache) store(key string, v dimValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) > 300000 {
		c.m = make(map[string]dimValue)
	}
	c.m[key] = v
}

type dimJob struct {
	index int
	key   string
	path  string
	ext   string
}

func (c *DimCache) Fill(entries []fsutil.Entry, budget time.Duration) {
	var jobs []dimJob
	for i := range entries {
		e := &entries[i]
		if e.Kind != fsutil.KindImage || !dimExts[e.Ext] {
			continue
		}
		key := dimKey(*e)
		if v, ok := c.lookup(key); ok {
			e.W, e.H = v.w, v.h
			continue
		}
		jobs = append(jobs, dimJob{index: i, key: key, path: e.Path, ext: e.Ext})
	}
	if len(jobs) == 0 {
		return
	}
	queue := make(chan dimJob)
	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range queue {
				w, h := ImageSize(job.path, job.ext)
				c.store(job.key, dimValue{w, h})
			}
		}()
	}
	go func() {
		for _, job := range jobs {
			queue <- job
		}
		close(queue)
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(budget):
	}
	for _, job := range jobs {
		if v, ok := c.lookup(job.key); ok {
			entries[job.index].W, entries[job.index].H = v.w, v.h
		}
	}
}
