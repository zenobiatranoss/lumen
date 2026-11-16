package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"

	"lumen/internal/config"
	"lumen/internal/fsutil"
)

type opRequest struct {
	Action  string   `json:"action"`
	Path    string   `json:"path"`
	Paths   []string `json:"paths"`
	Target  string   `json:"target"`
	Name    string   `json:"name"`
	Format  string   `json:"format"`
	Mode    string   `json:"mode"`
	Find    string   `json:"find"`
	Replace string   `json:"replace"`
}

type opItem struct {
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	Error string `json:"error,omitempty"`
}

type opResponse struct {
	Results []opItem `json:"results"`
	Count   int      `json:"count"`
}

func (s *Server) operate(w http.ResponseWriter, r *http.Request) {
	if !s.writable(w) {
		return
	}
	var req opRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "the request could not be read")
		return
	}
	paths := make([]string, 0, len(req.Paths)+1)
	for _, p := range append(req.Paths, req.Path) {
		if p == "" {
			continue
		}
		clean, err := s.cleanPath(p)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		paths = append(paths, clean)
	}
	if len(paths) > 5000 {
		writeError(w, http.StatusBadRequest, "too many items in one request")
		return
	}
	paths = slices.Compact(paths)
	idx := s.opts.Index
	var resp opResponse
	each := func(fn func(p string) (string, error)) {
		for _, p := range paths {
			to, err := fn(p)
			item := opItem{From: p, To: to}
			if err != nil {
				item.Error = friendly(err)
			}
			resp.Results = append(resp.Results, item)
		}
	}
	targetDir := func() (string, bool) {
		dir, err := s.cleanPath(req.Target)
		if err != nil || req.Target == "" || !isDir(dir) {
			writeError(w, http.StatusBadRequest, "the destination folder does not exist")
			return "", false
		}
		return dir, true
	}
	switch req.Action {
	case "rename":
		if len(paths) != 1 {
			writeError(w, http.StatusBadRequest, "choose one item to rename")
			return
		}
		each(func(p string) (string, error) {
			to, err := fsutil.Rename(p, req.Name)
			if err == nil && to != p {
				idx.Remove(p)
				idx.Add(to)
			}
			return to, err
		})
	case "mkdir":
		if len(paths) != 1 {
			writeError(w, http.StatusBadRequest, "choose one parent folder")
			return
		}
		each(func(p string) (string, error) {
			to, err := fsutil.Mkdir(p, req.Name)
			if err == nil {
				idx.Add(to)
			}
			return to, err
		})
	case "trash":
		each(func(p string) (string, error) {
			err := fsutil.MoveToTrash(p)
			if err == nil {
				idx.Remove(p)
			}
			return "", err
		})
	case "delete":
		each(func(p string) (string, error) {
			var err error
			if fsutil.InTrash(p) {
				err = fsutil.DeleteTrashed(p)
			} else {
				err = fsutil.Remove(p)
			}
			if err == nil {
				idx.Remove(p)
			}
			return "", err
		})
	case "copy":
		dir, ok := targetDir()
		if !ok {
			return
		}
		each(func(p string) (string, error) {
			to, err := fsutil.Copy(p, dir)
			if err == nil {
				idx.Add(to)
			}
			return to, err
		})
	case "move":
		dir, ok := targetDir()
		if !ok {
			return
		}
		each(func(p string) (string, error) {
			to, err := fsutil.Move(p, dir)
			if err == nil && to != p {
				idx.Remove(p)
				idx.Add(to)
			}
			return to, err
		})
	case "restore":
		each(func(p string) (string, error) {
			to, err := fsutil.RestoreTrash(p)
			if err == nil {
				idx.Add(to)
			}
			return to, err
		})
	case "emptytrash":
		n, err := fsutil.EmptyTrash()
		if err != nil {
			fail(w, err)
			return
		}
		resp.Count = n
	case "compress", "extract", "newfile", "duplicate", "symlink", "chmod", "bulkrename":
		if len(paths) == 0 {
			writeError(w, http.StatusBadRequest, "nothing is selected")
			return
		}
		resp.Results = s.extraOps(req, paths)
	default:
		writeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	if resp.Results == nil {
		resp.Results = []opItem{}
	}
	for _, item := range resp.Results {
		if item.Error == "" {
			resp.Count++
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) configGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.opts.Config.Get())
}

func (s *Server) configPut(w http.ResponseWriter, r *http.Request) {
	var incoming config.Config
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "the settings could not be read")
		return
	}
	before := s.opts.Config.Get()
	if err := s.opts.Config.Replace(incoming, s.opts.Home); err != nil {
		fail(w, err)
		return
	}
	after := s.opts.Config.Get()
	if !slices.Equal(before.IndexRoots, after.IndexRoots) || !slices.Equal(before.SkipDirs, after.SkipDirs) {
		s.opts.Index.Configure(after.IndexRoots, after.SkipDirs)
		s.opts.Index.Rebuild()
	}
	writeJSON(w, http.StatusOK, after)
}

func (s *Server) openExternal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
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
	if _, err := os.Lstat(path); err != nil {
		fail(w, err)
		return
	}
	var cmd *exec.Cmd
	if req.Mode == "reveal" {
		if dolphin, err := exec.LookPath("dolphin"); err == nil {
			cmd = exec.Command(dolphin, "--select", path)
		} else {
			cmd = exec.Command("xdg-open", filepath.Dir(path))
		}
	} else {
		cmd = exec.Command("xdg-open", path)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not launch the default application")
		return
	}
	go cmd.Wait()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) reindex(w http.ResponseWriter, r *http.Request) {
	s.opts.Index.Rebuild()
	writeJSON(w, http.StatusOK, s.opts.Index.Status())
}
