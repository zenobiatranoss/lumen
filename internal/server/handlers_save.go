package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"unicode/utf8"
)

const saveLimit = 8 << 20

func (s *Server) save(w http.ResponseWriter, r *http.Request) {
	if !s.writable(w) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mtime   int64  `json:"mtime"`
		Force   bool   `json:"force"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, saveLimit+1<<16)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "the file is too large to save from here")
		return
	}
	if len(req.Content) > saveLimit || !utf8.ValidString(req.Content) {
		writeError(w, http.StatusBadRequest, "the content cannot be saved")
		return
	}
	path, err := s.cleanPath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	info, err := os.Stat(path)
	if err != nil {
		fail(w, err)
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "that path is a folder")
		return
	}
	if !req.Force && req.Mtime != 0 && info.ModTime().UnixMilli() != req.Mtime {
		writeError(w, http.StatusConflict, "the file was changed by another program since you opened it")
		return
	}
	if err := writeAtomic(path, []byte(req.Content), info.Mode().Perm()); err != nil {
		fail(w, err)
		return
	}
	after, err := os.Stat(path)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"mtime": strconv.FormatInt(after.ModTime().UnixMilli(), 10),
		"size":  strconv.FormatInt(after.Size(), 10),
	})
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lumen-save-*")
	if err != nil {
		return os.WriteFile(path, data, perm)
	}
	name := tmp.Name()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(perm)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		os.Remove(name)
		return os.WriteFile(path, data, perm)
	}
	return nil
}
