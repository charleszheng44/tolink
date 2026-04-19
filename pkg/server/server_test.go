package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/charleszheng44/tolink/pkg/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(filepath.Join(dir, "links.json"))
	if err != nil {
		t.Fatal(err)
	}
	return New(s, []byte("<html>admin</html>"))
}

func TestKnownShortcutRedirect(t *testing.T) {
	sv := newTestServer(t)
	if err := sv.s.Set("gh", "https://github.com"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/gh", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "https://github.com" {
		t.Errorf("expected Location https://github.com, got %q", got)
	}
}

func TestUnknownShortcutRedirectsToAdmin(t *testing.T) {
	sv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/nosuchshortcut", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/.tolink/" {
		t.Errorf("expected Location /.tolink/, got %q", got)
	}
}

func TestAdminUIServed(t *testing.T) {
	sv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/.tolink/", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("unexpected Content-Type: %q", got)
	}
	if body := w.Body.String(); body != "<html>admin</html>" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestGetLinks(t *testing.T) {
	sv := newTestServer(t)
	sv.s.Set("gh", "https://github.com")

	req := httptest.NewRequest(http.MethodGet, "/.tolink/api/links", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var links map[string]string
	if err := json.NewDecoder(w.Body).Decode(&links); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if links["gh"] != "https://github.com" {
		t.Errorf("unexpected links: %v", links)
	}
}

func TestPostLink(t *testing.T) {
	sv := newTestServer(t)

	body, _ := json.Marshal(map[string]string{"shortcut": "gh", "url": "https://github.com"})
	req := httptest.NewRequest(http.MethodPost, "/.tolink/api/links", bytes.NewReader(body))
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if url, ok := sv.s.Get("gh"); !ok || url != "https://github.com" {
		t.Errorf("link not persisted: ok=%v url=%q", ok, url)
	}
}

func TestDeleteLink(t *testing.T) {
	sv := newTestServer(t)
	sv.s.Set("gh", "https://github.com")

	req := httptest.NewRequest(http.MethodDelete, "/.tolink/api/links/gh", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if _, ok := sv.s.Get("gh"); ok {
		t.Error("expected link to be deleted")
	}
}

func TestPostMissingFields(t *testing.T) {
	sv := newTestServer(t)

	for _, body := range []map[string]string{
		{"shortcut": "gh"},          // missing url
		{"url": "https://github.com"}, // missing shortcut
	} {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/.tolink/api/links", bytes.NewReader(b))
		w := httptest.NewRecorder()
		sv.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body=%v: expected 400, got %d", body, w.Code)
		}
	}
}

func TestPostInvalidURLScheme(t *testing.T) {
	sv := newTestServer(t)

	for _, badURL := range []string{"javascript:alert(1)", "ftp://example.com", "data:text/html,hi"} {
		body, _ := json.Marshal(map[string]string{"shortcut": "x", "url": badURL})
		req := httptest.NewRequest(http.MethodPost, "/.tolink/api/links", bytes.NewReader(body))
		w := httptest.NewRecorder()
		sv.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("url=%q: expected 400, got %d", badURL, w.Code)
		}
	}
}

func TestDeleteNonExistentShortcut(t *testing.T) {
	sv := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/.tolink/api/links/nosuch", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteCollectionPathReturnsBadRequest(t *testing.T) {
	sv := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/.tolink/api/links", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetSubPathNotAllowed(t *testing.T) {
	sv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/.tolink/api/links/foo", nil)
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPostSubPathNotAllowed(t *testing.T) {
	sv := newTestServer(t)

	body, _ := json.Marshal(map[string]string{"shortcut": "gh", "url": "https://github.com"})
	req := httptest.NewRequest(http.MethodPost, "/.tolink/api/links/foo", bytes.NewReader(body))
	w := httptest.NewRecorder()
	sv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
