package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"lumen/internal/config"
	"lumen/internal/fsutil"
	icontheme "lumen/internal/icons"
	"lumen/internal/media"
	"lumen/internal/search"
)

const cookieName = "lumen"

type Options struct {
	Home     string
	Config   *config.Store
	Index    *search.Index
	Thumbs   *media.Thumbs
	Web      fs.FS
	Token    string
	ReadOnly bool
	Version  string
}

type Server struct {
	opts     Options
	dims     *media.DimCache
	mux      *http.ServeMux
	lastSeen atomic.Int64
	hostsMu  sync.RWMutex
	hosts    map[string]bool
	previews previewCache
	icons    *icontheme.Resolver
}

func New(opts Options) *Server {
	s := &Server{
		opts:     opts,
		dims:     media.NewDimCache(),
		mux:      http.NewServeMux(),
		hosts:    map[string]bool{},
		previews: previewCache{items: map[string]previewEntry{}},
		icons:    icontheme.New(),
	}
	s.lastSeen.Store(time.Now().UnixNano())
	s.routes()
	return s
}

func (s *Server) SetPort(port int) {
	s.hostsMu.Lock()
	defer s.hostsMu.Unlock()
	s.hosts = map[string]bool{
		fmt.Sprintf("127.0.0.1:%d", port): true,
		fmt.Sprintf("localhost:%d", port): true,
	}
}

func (s *Server) IdleFor() time.Duration {
	return time.Since(time.Unix(0, s.lastSeen.Load()))
}

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /{$}", s.indexPage)
	m.Handle("GET /", noCache(http.FileServerFS(s.opts.Web)))

	m.HandleFunc("GET /api/ping", s.ping)
	m.HandleFunc("GET /api/meta", s.meta)
	m.HandleFunc("GET /api/list", s.list)
	m.HandleFunc("GET /api/search", s.search)
	m.HandleFunc("GET /api/preview", s.preview)
	m.HandleFunc("GET /api/places", s.places)
	m.HandleFunc("GET /api/info", s.info)
	m.HandleFunc("GET /api/dirsize", s.dirSize)
	m.HandleFunc("GET /api/trash", s.trashList)
	m.HandleFunc("GET /api/file", s.file)
	m.HandleFunc("GET /api/thumb", s.thumb)
	m.HandleFunc("GET /api/text", s.text)
	m.HandleFunc("GET /api/stream", s.stream)
	m.HandleFunc("GET /api/cache", s.cacheStats)
	m.HandleFunc("DELETE /api/cache", s.cacheClear)
	m.HandleFunc("GET /api/config", s.configGet)
	m.HandleFunc("PUT /api/config", s.configPut)
	m.HandleFunc("POST /api/op", s.operate)
	m.HandleFunc("POST /api/open", s.openExternal)
	m.HandleFunc("GET /api/checksum", s.checksum)
	m.HandleFunc("GET /api/openwith", s.openWithList)
	m.HandleFunc("POST /api/openwith", s.openWithRun)
	m.HandleFunc("GET /api/icon", s.themeIcon)
	m.HandleFunc("GET /api/themeicon", s.themeIcon)
	m.HandleFunc("GET /api/dev", s.devInfo)
	m.HandleFunc("POST /api/dev", s.devRun)
	m.HandleFunc("POST /api/crypt", s.crypt)
	m.HandleFunc("POST /api/terminal", s.terminal)
	m.HandleFunc("GET /api/sniff", s.sniff)
	m.HandleFunc("POST /api/run", s.run)
	m.HandleFunc("POST /api/save", s.save)
	m.HandleFunc("POST /api/reindex", s.reindex)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.lastSeen.Store(time.Now().UnixNano())
		if !s.hostAllowed(r.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if !s.authorized(r) {
				writeError(w, http.StatusUnauthorized, "this session is no longer valid, reopen Lumen from the terminal")
				return
			}
			if r.Method != http.MethodGet && !s.sameOrigin(r) {
				writeError(w, http.StatusForbidden, "cross-origin request blocked")
				return
			}
		}
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) hostAllowed(host string) bool {
	s.hostsMu.RLock()
	defer s.hostsMu.RUnlock()
	return s.hosts[strings.ToLower(host)]
}

func (s *Server) authorized(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.opts.Token)) == 1
}

func (s *Server) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

func (s *Server) indexPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if given := query.Get("t"); given != "" {
		if subtle.ConstantTimeCompare([]byte(given), []byte(s.opts.Token)) != 1 {
			http.Error(w, "invalid token", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    s.opts.Token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		query.Del("t")
		target := "/"
		if len(query) > 0 {
			target += "?" + query.Encode()
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, s.opts.Web, "index.html")
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, fs.ErrPermission):
		return http.StatusForbidden
	case errors.Is(err, fsutil.ErrExists):
		return http.StatusConflict
	case errors.Is(err, fsutil.ErrPassword):
		return http.StatusBadRequest
	case errors.Is(err, fsutil.ErrProtected), errors.Is(err, fsutil.ErrInside):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func friendly(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		switch {
		case errors.Is(pe.Err, fs.ErrNotExist):
			return "that item no longer exists"
		case errors.Is(pe.Err, fs.ErrPermission):
			return "permission denied"
		case errors.Is(pe.Err, syscall.ENOSPC):
			return "there is no space left on the device"
		case errors.Is(pe.Err, syscall.ENOTEMPTY):
			return "the folder is not empty"
		}
		return pe.Err.Error()
	}
	return err.Error()
}

func fail(w http.ResponseWriter, err error) {
	writeError(w, statusFor(err), friendly(err))
}

func (s *Server) cleanPath(p string) (string, error) {
	if p == "" {
		return s.opts.Home, nil
	}
	if strings.ContainsRune(p, 0) {
		return "", errors.New("invalid path")
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		p = filepath.Join(s.opts.Home, p[1:])
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("the path must be absolute")
	}
	return filepath.Clean(p), nil
}

func (s *Server) pathParam(r *http.Request) (string, error) {
	return s.cleanPath(r.URL.Query().Get("path"))
}

func (s *Server) writable(w http.ResponseWriter) bool {
	if s.opts.ReadOnly {
		writeError(w, http.StatusForbidden, "Lumen is running in read-only mode")
		return false
	}
	return true
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
