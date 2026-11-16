package search

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"lumen/internal/fsutil"
)

type Item struct {
	Path    string
	NameAt  int
	Size    int64
	ModTime int64
	Kind    fsutil.Kind
}

func (it Item) Name() string {
	return it.Path[it.NameAt:]
}

func (it Item) Entry() fsutil.Entry {
	name := it.Name()
	e := fsutil.Entry{
		Name:    name,
		Path:    it.Path,
		Kind:    it.Kind,
		Size:    it.Size,
		ModTime: it.ModTime,
	}
	if it.Kind != fsutil.KindFolder {
		e.Ext = fsutil.Ext(name)
		e.Native = fsutil.BrowserNative(e.Ext, it.Kind)
	}
	return e
}

func kindAndSize(info fs.FileInfo, name string) (fsutil.Kind, int64) {
	if info.Mode().IsRegular() {
		return fsutil.KindOf(fsutil.Ext(name)), info.Size()
	}
	return fsutil.KindOther, 0
}

func newItem(path string, size, mtime int64, kind fsutil.Kind) Item {
	return Item{Path: path, NameAt: strings.LastIndexByte(path, '/') + 1, Size: size, ModTime: mtime, Kind: kind}
}

type WalkOptions struct {
	Hidden    bool
	SkipDir   map[string]bool
	Workers   int
	SkipPaths map[string]bool
}

type walker struct {
	mu     sync.Mutex
	cond   *sync.Cond
	stack  []string
	active int
}

func newWalker() *walker {
	w := &walker{}
	w.cond = sync.NewCond(&w.mu)
	return w
}

func (w *walker) push(dirs ...string) {
	w.mu.Lock()
	w.stack = append(w.stack, dirs...)
	w.mu.Unlock()
	w.cond.Broadcast()
}

func (w *walker) next(ctx context.Context) (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for len(w.stack) == 0 && w.active > 0 && ctx.Err() == nil {
		w.cond.Wait()
	}
	if ctx.Err() != nil || len(w.stack) == 0 {
		w.cond.Broadcast()
		return "", false
	}
	dir := w.stack[len(w.stack)-1]
	w.stack = w.stack[:len(w.stack)-1]
	w.active++
	return dir, true
}

func (w *walker) finish() {
	w.mu.Lock()
	w.active--
	w.mu.Unlock()
	w.cond.Broadcast()
}

func Walk(ctx context.Context, roots []string, opts WalkOptions, visit func(Item)) {
	if opts.Workers <= 0 {
		opts.Workers = 8
	}
	w := newWalker()
	w.push(roots...)
	stop := context.AfterFunc(ctx, func() { w.cond.Broadcast() })
	defer stop()
	var wg sync.WaitGroup
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				dir, ok := w.next(ctx)
				if !ok {
					return
				}
				scanDir(ctx, dir, opts, w, visit)
				w.finish()
			}
		}()
	}
	wg.Wait()
}

func scanDir(ctx context.Context, dir string, opts WalkOptions, w *walker, visit func(Item)) {
	entries, err := os.ReadDir(dir)
	if err != nil && len(entries) == 0 {
		return
	}
	var subdirs []string
	for _, de := range entries {
		if ctx.Err() != nil {
			return
		}
		name := de.Name()
		if !opts.Hidden && strings.HasPrefix(name, ".") {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(dir, name)
		if de.IsDir() {
			if opts.SkipDir[name] || opts.SkipPaths[full] {
				continue
			}
			visit(newItem(full, 0, info.ModTime().UnixMilli(), fsutil.KindFolder))
			subdirs = append(subdirs, full)
			continue
		}
		kind, size := kindAndSize(info, name)
		visit(newItem(full, size, info.ModTime().UnixMilli(), kind))
	}
	if len(subdirs) > 0 {
		w.push(subdirs...)
	}
}
