package search

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Status struct {
	Count    int       `json:"count"`
	Building bool      `json:"building"`
	Ready    bool      `json:"ready"`
	BuiltAt  time.Time `json:"builtAt"`
}

type Index struct {
	mu       sync.RWMutex
	items    []Item
	roots    []string
	skip     map[string]bool
	building bool
	ready    bool
	builtAt  time.Time
	cancel   context.CancelFunc
}

func NewIndex() *Index {
	return &Index{skip: map[string]bool{}}
}

func (ix *Index) Configure(roots, skipDirs []string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.roots = append([]string(nil), roots...)
	ix.skip = make(map[string]bool, len(skipDirs))
	for _, s := range skipDirs {
		ix.skip[s] = true
	}
}

func (ix *Index) Roots() []string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return append([]string(nil), ix.roots...)
}

func (ix *Index) Skip() map[string]bool {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := make(map[string]bool, len(ix.skip))
	for k, v := range ix.skip {
		out[k] = v
	}
	return out
}

func (ix *Index) Status() Status {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return Status{Count: len(ix.items), Building: ix.building, Ready: ix.ready, BuiltAt: ix.builtAt}
}

func (ix *Index) Rebuild() {
	ix.mu.Lock()
	if ix.cancel != nil {
		ix.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ix.cancel = cancel
	ix.building = true
	roots := append([]string(nil), ix.roots...)
	skip := ix.skip
	ix.mu.Unlock()

	go func() {
		var mu sync.Mutex
		var items []Item
		Walk(ctx, roots, WalkOptions{SkipDir: skip}, func(it Item) {
			mu.Lock()
			items = append(items, it)
			mu.Unlock()
		})
		if ctx.Err() != nil {
			return
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
		ix.mu.Lock()
		ix.items = items
		ix.building = false
		ix.ready = true
		ix.builtAt = time.Now()
		ix.mu.Unlock()
	}()
}

func (ix *Index) Stop() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if ix.cancel != nil {
		ix.cancel()
	}
}

func (ix *Index) Covers(path string) bool {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	if !ix.ready {
		return false
	}
	for _, r := range ix.roots {
		if path == r || strings.HasPrefix(path, strings.TrimSuffix(r, "/")+"/") {
			return true
		}
	}
	return false
}

func inScope(path, scope string) bool {
	if scope == "" || scope == "/" {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(scope, "/")+"/")
}

func (ix *Index) Search(q Query, scope string, limit int) ([]Item, int) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var out []Item
	total := 0
	for _, it := range ix.items {
		if !inScope(it.Path, scope) || !q.Match(it) {
			continue
		}
		total++
		if len(out) < limit {
			out = append(out, it)
		}
	}
	return out, total
}

func (ix *Index) Remove(prefix string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	kept := ix.items[:0]
	for _, it := range ix.items {
		if it.Path == prefix || strings.HasPrefix(it.Path, prefix+"/") {
			continue
		}
		kept = append(kept, it)
	}
	ix.items = kept
}

func (ix *Index) Add(path string) {
	ix.mu.RLock()
	skip := ix.skip
	ready := ix.ready
	ix.mu.RUnlock()
	if !ready {
		return
	}
	info, err := os.Lstat(path)
	if err != nil || strings.HasPrefix(filepath.Base(path), ".") {
		return
	}
	var added []Item
	var mu sync.Mutex
	collect := func(it Item) {
		mu.Lock()
		added = append(added, it)
		mu.Unlock()
	}
	if info.IsDir() {
		collect(newItem(path, 0, info.ModTime().UnixMilli(), "folder"))
		Walk(context.Background(), []string{path}, WalkOptions{SkipDir: skip, Workers: 4}, collect)
	} else {
		kind, size := kindAndSize(info, filepath.Base(path))
		collect(newItem(path, size, info.ModTime().UnixMilli(), kind))
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.items = append(ix.items, added...)
	sort.Slice(ix.items, func(i, j int) bool { return ix.items[i].Path < ix.items[j].Path })
}

func LiveSearch(ctx context.Context, root string, q Query, hidden bool, skip, skipPaths map[string]bool, limit int) ([]Item, int) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	var out []Item
	total := 0
	Walk(ctx, []string{root}, WalkOptions{Hidden: hidden, SkipDir: skip, SkipPaths: skipPaths}, func(it Item) {
		if !q.Match(it) || !q.MatchContent(it) {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		total++
		if len(out) < limit {
			out = append(out, it)
		}
		if total >= limit*4 {
			cancel()
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, total
}
