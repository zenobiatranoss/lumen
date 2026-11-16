package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type sniffResult struct {
	Text       bool   `json:"text"`
	Exec       string `json:"exec"`
	Executable bool   `json:"executable"`
	Library    bool   `json:"library"`
	Size       int64  `json:"size"`
}

func sniffFile(path string) (sniffResult, error) {
	var res sniffResult
	f, err := os.Open(path)
	if err != nil {
		return res, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return res, err
	}
	res.Size = info.Size()
	res.Executable = info.Mode()&0o111 != 0
	data, err := io.ReadAll(io.LimitReader(f, 8192))
	if err != nil {
		return res, err
	}
	lower := strings.ToLower(filepath.Base(path))
	switch {
	case bytes.HasPrefix(data, []byte("\x7fELF")):
		res.Exec = "elf"
		res.Library = strings.HasSuffix(lower, ".so") || strings.Contains(lower, ".so.")
	case bytes.HasPrefix(data, []byte("MZ")) && bytes.IndexByte(data, 0) >= 0:
		res.Exec = "pe"
		res.Library = strings.HasSuffix(lower, ".dll") || strings.HasSuffix(lower, ".sys")
	}
	res.Text = res.Exec == "" && looksLikeText(data)
	return res, nil
}

func looksLikeText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	trimmed := data
	for i := 0; i < 3 && len(trimmed) > 0 && !utf8.Valid(trimmed); i++ {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if utf8.Valid(trimmed) {
		return true
	}
	control := 0
	for _, b := range data {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' && b != '\f' && b != 0x1b {
			control++
		}
	}
	return control*100 < len(data)
}

func (s *Server) sniff(w http.ResponseWriter, r *http.Request) {
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
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "that path is a folder")
		return
	}
	res, err := sniffFile(path)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) run(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string `json:"path"`
		Chmod bool   `json:"chmod"`
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
	res, err := sniffFile(path)
	if err != nil {
		fail(w, err)
		return
	}
	dir := filepath.Dir(path)
	switch {
	case res.Text || res.Exec == "":
		writeError(w, http.StatusBadRequest, "this file is not a runnable program")
	case res.Library:
		writeError(w, http.StatusBadRequest, "this is a library, not a program")
	case res.Exec == "pe":
		wine, err := exec.LookPath("wine")
		if err != nil {
			writeError(w, http.StatusNotFound, "this is a Windows program, install Wine to run it: sudo apt install wine")
			return
		}
		if err := startDetached(dir, wine, path); err != nil {
			writeError(w, http.StatusInternalServerError, "could not start the program")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		if !res.Executable {
			if !req.Chmod {
				writeError(w, http.StatusConflict, "this file is not marked as executable")
				return
			}
			if !s.writable(w) {
				return
			}
			info, err := os.Stat(path)
			if err == nil {
				err = os.Chmod(path, info.Mode().Perm()|0o111)
			}
			if err != nil {
				fail(w, err)
				return
			}
		}
		if err := startDetached(dir, path); err != nil {
			writeError(w, http.StatusInternalServerError, "could not start the program")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
