package server

import (
	"context"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"lumen/internal/fsutil"
	"lumen/internal/media"
	"lumen/internal/search"
)

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) meta(w http.ResponseWriter, r *http.Request) {
	tools := s.opts.Thumbs.Tools()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":  s.opts.Version,
		"home":     s.opts.Home,
		"readOnly": s.opts.ReadOnly,
		"ffmpeg":   tools.FFmpeg != "",
		"ffprobe":  tools.FFprobe != "",
		"pdf":      tools.PDFToPPM != "",
		"convert":  tools.Convert != "",
		"index":    s.opts.Index.Status(),
	})
}

type listResponse struct {
	Path     string         `json:"path"`
	Parent   string         `json:"parent"`
	Entries  []fsutil.Entry `json:"entries"`
	Writable bool           `json:"writable"`
	Total    uint64         `json:"total"`
	Free     uint64         `json:"free"`
	Trash    bool           `json:"trash"`
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		fail(w, err)
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "that path is not a folder")
		return
	}
	hidden := r.URL.Query().Get("hidden") == "1"
	entries, err := fsutil.ListDir(path, hidden)
	if err != nil {
		fail(w, err)
		return
	}
	s.dims.Fill(entries, 1200*time.Millisecond)
	parent := filepath.Dir(path)
	if parent == path {
		parent = ""
	}
	total, free := fsutil.FreeSpace(path)
	writeJSON(w, http.StatusOK, listResponse{
		Path:     path,
		Parent:   parent,
		Entries:  entries,
		Writable: !s.opts.ReadOnly && syscall.Access(path, 2) == nil,
		Total:    total,
		Free:     free,
		Trash:    fsutil.InTrash(filepath.Join(path, "x")),
	})
}

type searchResponse struct {
	Query     string         `json:"query"`
	Results   []fsutil.Entry `json:"results"`
	Total     int            `json:"total"`
	Truncated bool           `json:"truncated"`
	TookMs    int64          `json:"tookMs"`
	Live      bool           `json:"live"`
	Indexing  bool           `json:"indexing"`
}

const searchLimit = 3000

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	parsed, err := search.Parse(query.Get("q"), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dir, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	hidden := query.Get("hidden") == "1"
	inFiles := query.Get("content") == "1"
	if inFiles {
		parsed.Contains, parsed.Terms = parsed.Terms, nil
	}
	status := s.opts.Index.Status()
	resp := searchResponse{Query: query.Get("q"), Results: []fsutil.Entry{}, Indexing: status.Building}
	if parsed.Empty() {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	scope := dir
	switch query.Get("scope") {
	case "everywhere":
		scope = ""
	case "system":
		scope = "/"
	}
	systemWide := scope == "/"
	started := time.Now()
	var items []search.Item
	if !hidden && !inFiles && !systemWide && status.Ready && (scope == "" || s.opts.Index.Covers(scope)) {
		items, resp.Total = s.opts.Index.Search(parsed, scope, searchLimit)
	} else {
		resp.Live = true
		root := scope
		if root == "" {
			root = s.opts.Home
		}
		timeout := 20 * time.Second
		var skipPaths map[string]bool
		if systemWide {
			timeout = 45 * time.Second
			skipPaths = systemSkipPaths()
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		items, resp.Total = search.LiveSearch(ctx, root, parsed, hidden, s.opts.Index.Skip(), skipPaths, searchLimit)
	}
	resp.Truncated = resp.Total > len(items)
	for _, it := range items {
		resp.Results = append(resp.Results, it.Entry())
	}
	s.dims.Fill(resp.Results, 900*time.Millisecond)
	resp.TookMs = time.Since(started).Milliseconds()
	writeJSON(w, http.StatusOK, resp)
}

type previewEntry struct {
	mtime  int64
	images []string
	count  int
}

type previewCache struct {
	mu    sync.Mutex
	items map[string]previewEntry
}

func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		writeJSON(w, http.StatusOK, map[string]any{"images": []string{}, "count": 0})
		return
	}
	mtime := info.ModTime().UnixNano()
	s.previews.mu.Lock()
	cached, ok := s.previews.items[path]
	s.previews.mu.Unlock()
	if !ok || cached.mtime != mtime {
		images, count := fsutil.FirstImages(path, 4)
		cached = previewEntry{mtime: mtime, images: images, count: count}
		s.previews.mu.Lock()
		if len(s.previews.items) > 4000 {
			s.previews.items = map[string]previewEntry{}
		}
		s.previews.items[path] = cached
		s.previews.mu.Unlock()
	}
	if cached.images == nil {
		cached.images = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"images": cached.images, "count": cached.count})
}

func (s *Server) places(w http.ResponseWriter, r *http.Request) {
	trash, _ := fsutil.ListTrash()
	writeJSON(w, http.StatusOK, map[string]any{
		"home":       s.opts.Home,
		"places":     fsutil.Places(s.opts.Home),
		"favorites":  s.opts.Config.Get().Favorites,
		"drives":     fsutil.Drives(),
		"trashCount": len(trash),
	})
}

type infoResponse struct {
	fsutil.Entry
	Dir      string       `json:"dir"`
	Mode     string       `json:"mode"`
	Owner    string       `json:"owner,omitempty"`
	Group    string       `json:"group,omitempty"`
	Accessed int64        `json:"accessed,omitempty"`
	Changed  int64        `json:"changed,omitempty"`
	MIME     string       `json:"mime"`
	Target   string       `json:"target,omitempty"`
	Items    int          `json:"items,omitempty"`
	EXIF     *media.EXIF  `json:"exif,omitempty"`
	Probe    *media.Probe `json:"probe,omitempty"`
}

func (s *Server) info(w http.ResponseWriter, r *http.Request) {
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
	resp := infoResponse{Entry: entry, Dir: filepath.Dir(path), MIME: contentType(entry)}
	if info, err := os.Stat(path); err == nil {
		resp.Mode = info.Mode().String()
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			resp.Accessed = time.Unix(int64(st.Atim.Sec), 0).UnixMilli()
			resp.Changed = time.Unix(int64(st.Ctim.Sec), 0).UnixMilli()
			if u, err := user.LookupId(strconv.FormatUint(uint64(st.Uid), 10)); err == nil {
				resp.Owner = u.Username
			}
			if g, err := user.LookupGroupId(strconv.FormatUint(uint64(st.Gid), 10)); err == nil {
				resp.Group = g.Name
			}
		}
	}
	if entry.Link {
		resp.Target, _ = os.Readlink(path)
	}
	switch entry.Kind {
	case fsutil.KindFolder:
		if items, err := os.ReadDir(path); err == nil {
			resp.Items = len(items)
		}
	case fsutil.KindImage:
		resp.W, resp.H = media.ImageSize(path, entry.Ext)
		if ex, ok := media.ReadEXIF(path); ok && !ex.Empty() {
			resp.EXIF = &ex
		}
	case fsutil.KindVideo, fsutil.KindAudio:
		if p, err := s.opts.Thumbs.Tools().Probe(r.Context(), path); err == nil {
			resp.Probe = p
			resp.W, resp.H = p.Width, p.Height
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) dirSize(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	stats, err := fsutil.DirStats(ctx, path)
	if err != nil && ctx.Err() == nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"size":     stats.Size,
		"files":    stats.Files,
		"folders":  stats.Folders,
		"complete": ctx.Err() == nil,
	})
}

func (s *Server) trashList(w http.ResponseWriter, r *http.Request) {
	items, err := fsutil.ListTrash()
	if err != nil {
		fail(w, err)
		return
	}
	if items == nil {
		items = []fsutil.TrashItem{}
	}
	entries := make([]fsutil.Entry, len(items))
	for i := range items {
		entries[i] = items[i].Entry
	}
	s.dims.Fill(entries, 600*time.Millisecond)
	for i := range items {
		items[i].Entry = entries[i]
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": items})
}

func systemSkipPaths() map[string]bool {
	skip := map[string]bool{}
	for _, p := range []string{"/proc", "/sys", "/dev", "/run", "/snap", "/lost+found", "/var/lib/docker", "/var/lib/flatpak", "/var/cache"} {
		skip[p] = true
	}
	for _, p := range fsutil.SlowMounts() {
		skip[p] = true
	}
	return skip
}
