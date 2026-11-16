package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"lumen/internal/dev"
)

func (s *Server) devInfo(w http.ResponseWriter, r *http.Request) {
	path, err := s.pathParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dev.Describe(path))
}

func (s *Server) devRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Args string `json:"args"`
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
	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	res, err := dev.Run(ctx, path, req.Args)
	if err != nil {
		var missing *dev.MissingError
		if errors.As(err, &missing) {
			writeError(w, http.StatusUnprocessableEntity, missing.Error())
			return
		}
		if errors.Is(err, dev.ErrUnsupported) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
