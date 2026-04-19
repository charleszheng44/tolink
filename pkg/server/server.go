package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/charleszheng44/tolink/pkg/store"
)

type Server struct {
	store *store.Store
	ui    []byte
}

func New(s *store.Store, ui []byte) *Server {
	return &Server{store: s, ui: ui}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/.tolink/") {
		s.handleAdmin(w, r)
		return
	}
	shortcut := strings.TrimPrefix(path, "/")
	if url, ok := s.store.Get(shortcut); ok {
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/.tolink/", http.StatusFound)
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/.tolink/":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(s.ui)
	case strings.HasPrefix(path, "/.tolink/api/links"):
		s.handleLinks(w, r)
	default:
		http.NotFound(w, r)
	}
}

type linkRequest struct {
	Shortcut string `json:"shortcut"`
	URL      string `json:"url"`
}

func (s *Server) handleLinks(w http.ResponseWriter, r *http.Request) {
	shortcut := strings.TrimPrefix(r.URL.Path, "/.tolink/api/links")
	shortcut = strings.TrimPrefix(shortcut, "/")

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.store.List())

	case http.MethodPost:
		var req linkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Shortcut == "" || req.URL == "" {
			http.Error(w, "shortcut and url required", http.StatusBadRequest)
			return
		}
		if err := s.store.Set(req.Shortcut, req.URL); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if shortcut == "" {
			http.Error(w, "shortcut required", http.StatusBadRequest)
			return
		}
		if err := s.store.Delete(shortcut); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
