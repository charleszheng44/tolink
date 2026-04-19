package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/charleszheng44/tolink/pkg/server"
	"github.com/charleszheng44/tolink/pkg/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "links.json"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return s
}

func TestRedirect_KnownShortcut(t *testing.T) {
	s := newStore(t)
	s.Set("gh", "https://github.com")
	srv := server.New(s, []byte("<html>admin</html>"))

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://github.com" {
		t.Fatalf("expected location https://github.com, got %q", loc)
	}
}

func TestRedirect_UnknownShortcut(t *testing.T) {
	srv := server.New(newStore(t), []byte("<html>admin</html>"))
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/.tolink/" {
		t.Fatalf("expected redirect to /.tolink/, got %q", loc)
	}
}

func TestAdminUI(t *testing.T) {
	srv := server.New(newStore(t), []byte("<html>admin</html>"))
	req := httptest.NewRequest(http.MethodGet, "/.tolink/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if body := w.Body.String(); body != "<html>admin</html>" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestAPIGetLinks(t *testing.T) {
	s := newStore(t)
	s.Set("gh", "https://github.com")
	srv := server.New(s, []byte("<html>admin</html>"))

	req := httptest.NewRequest(http.MethodGet, "/.tolink/api/links", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var links map[string]string
	if err := json.NewDecoder(w.Body).Decode(&links); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if links["gh"] != "https://github.com" {
		t.Fatalf("unexpected links: %v", links)
	}
}

func TestAPIPostLink(t *testing.T) {
	srv := server.New(newStore(t), []byte("<html>admin</html>"))
	body, _ := json.Marshal(map[string]string{"shortcut": "gh", "url": "https://github.com"})
	req := httptest.NewRequest(http.MethodPost, "/.tolink/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/.tolink/api/links", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	var links map[string]string
	json.NewDecoder(w2.Body).Decode(&links)
	if links["gh"] != "https://github.com" {
		t.Fatalf("link not persisted: %v", links)
	}
}

func TestAPIDeleteLink(t *testing.T) {
	s := newStore(t)
	s.Set("gh", "https://github.com")
	srv := server.New(s, []byte("<html>admin</html>"))

	req := httptest.NewRequest(http.MethodDelete, "/.tolink/api/links/gh", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/.tolink/api/links", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	var links map[string]string
	json.NewDecoder(w2.Body).Decode(&links)
	if _, ok := links["gh"]; ok {
		t.Fatal("link should have been deleted")
	}
}

func TestAPIPostLink_MissingFields(t *testing.T) {
	srv := server.New(newStore(t), []byte("<html>admin</html>"))
	body, _ := json.Marshal(map[string]string{"shortcut": "gh"})
	req := httptest.NewRequest(http.MethodPost, "/.tolink/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
