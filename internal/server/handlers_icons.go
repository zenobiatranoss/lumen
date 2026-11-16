package server

import (
	"io"
	"net/http"
	"os"
	"strconv"
)

func (s *Server) themeIcon(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	size := 64
	if given, err := strconv.Atoi(r.URL.Query().Get("size")); err == nil && given > 0 && given <= 1024 {
		size = given
	}
	path, typ, ok := s.icons.Lookup(name, size)
	if !ok {
		writeError(w, http.StatusNotFound, "that icon is not available")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "that icon is not available")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "that icon is not available")
		return
	}
	h := w.Header()
	h.Set("Content-Type", typ)
	h.Set("Cache-Control", "private, max-age=86400")
	if typ == "image/svg+xml" {
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, io.LimitReader(f, 4<<20))
}
