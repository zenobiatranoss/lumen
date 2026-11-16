package server

import (
	"encoding/json"
	"net/http"

	"lumen/internal/fsutil"
)

func (s *Server) crypt(w http.ResponseWriter, r *http.Request) {
	if !s.writable(w) {
		return
	}
	var req struct {
		Path     string `json:"path"`
		Password string `json:"password"`
		Action   string `json:"action"`
		Trash    bool   `json:"trash"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "the request could not be read")
		return
	}
	path, err := s.cleanPath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Password) < 4 {
		writeError(w, http.StatusBadRequest, "choose a password with at least 4 characters")
		return
	}
	var out string
	var warning string
	switch req.Action {
	case "encrypt":
		out, err = fsutil.Encrypt(path, req.Password)
		if err == nil {
			s.opts.Index.Add(out)
			if req.Trash {
				if trashErr := fsutil.MoveToTrash(path); trashErr != nil {
					warning = "The encrypted copy was created, but the original could not be moved to the trash."
				} else {
					s.opts.Index.Remove(path)
				}
			}
		}
	case "decrypt":
		out, err = fsutil.Decrypt(path, req.Password)
		if err == nil {
			s.opts.Index.Add(out)
		}
	default:
		writeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "to": out, "warning": warning})
}
