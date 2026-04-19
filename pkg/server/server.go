package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/charleszheng44/tolink/pkg/store"
)

// Server routes HTTP requests: admin UI under /.tolink/, redirects for known shortcuts.
type Server struct {
	s  *store.Store
	ui []byte
}

// New creates a Server with the given store and embedded UI bytes.
func New(s *store.Store, ui []byte) *Server {
	return &Server{s: s, ui: ui}
}

func (sv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/.tolink/") {
		sv.adminHandler(w, r)
		return
	}
	shortcut := strings.TrimPrefix(r.URL.Path, "/")
	if shortcut != "" {
		if url, ok := sv.s.Get(shortcut); ok {
			http.Redirect(w, r, url, http.StatusFound)
			return
		}
	}
	http.Redirect(w, r, "/.tolink/", http.StatusFound)
}

func (sv *Server) adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/.tolink/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(sv.ui); err != nil {
			log.Printf("write ui: %v", err)
		}
		return
	}
	if r.URL.Path == "/.tolink/api/links" || strings.HasPrefix(r.URL.Path, "/.tolink/api/links/") {
		sv.handleLinks(w, r)
		return
	}
	http.NotFound(w, r)
}

func (sv *Server) handleLinks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		links := sv.s.List()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(links); err != nil {
			log.Printf("encode links: %v", err)
		}
	case http.MethodPost:
		var body struct {
			Shortcut string `json:"shortcut"`
			URL      string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.Shortcut == "" || body.URL == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		u, err := url.Parse(body.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			http.Error(w, "url must start with http:// or https://", http.StatusBadRequest)
			return
		}
		if err := sv.s.Set(body.Shortcut, body.URL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		shortcut := strings.TrimPrefix(r.URL.Path, "/.tolink/api/links/")
		if shortcut == "" {
			http.Error(w, "missing shortcut", http.StatusBadRequest)
			return
		}
		if err := sv.s.Delete(shortcut); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, "shortcut not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
