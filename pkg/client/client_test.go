package client_test

import (
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/charleszheng44/tolink/pkg/client"
	"github.com/charleszheng44/tolink/pkg/server"
	"github.com/charleszheng44/tolink/pkg/store"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "links.json"))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(server.New(st, nil)), st
}

func TestHTTPClientHappyPath(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	c, err := client.New(srv.URL, filepath.Join(t.TempDir(), "links.json"), false)
	if err != nil {
		t.Fatal(err)
	}

	links, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 0 {
		t.Errorf("expected empty list, got %v", links)
	}

	if err := c.Set("gh", "https://github.com"); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("gh")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://github.com" {
		t.Errorf("expected https://github.com, got %q", got)
	}

	links, err = c.List()
	if err != nil {
		t.Fatal(err)
	}
	if links["gh"] != "https://github.com" {
		t.Errorf("expected gh in list, got %v", links)
	}

	if err := c.Delete("gh"); err != nil {
		t.Fatal(err)
	}

	_, err = c.Get("gh")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestHTTPClientErrNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	c, err := client.New(srv.URL, filepath.Join(t.TempDir(), "links.json"), false)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Delete("missing"); !errors.Is(err, client.ErrNotFound) {
		t.Errorf("Delete missing: expected ErrNotFound, got %v", err)
	}

	if _, err := c.Get("missing"); !errors.Is(err, client.ErrNotFound) {
		t.Errorf("Get missing: expected ErrNotFound, got %v", err)
	}
}

func TestHTTPClientValidationError(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	c, err := client.New(srv.URL, filepath.Join(t.TempDir(), "links.json"), false)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Set("gh", "ftp://bad-scheme.com"); err == nil {
		t.Error("expected error for invalid URL scheme")
	}
}

func TestFileClientHappyPath(t *testing.T) {
	srv, _ := newTestServer(t)
	srvURL := srv.URL
	srv.Close()

	dataPath := filepath.Join(t.TempDir(), "links.json")
	c, err := client.New(srvURL, dataPath, false)
	if err != nil {
		t.Fatal(err)
	}

	links, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 0 {
		t.Errorf("expected empty list, got %v", links)
	}

	if err := c.Set("gl", "https://gitlab.com"); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("gl")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://gitlab.com" {
		t.Errorf("expected https://gitlab.com, got %q", got)
	}

	if err := c.Delete("gl"); err != nil {
		t.Fatal(err)
	}

	_, err = c.Get("gl")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestFileClientErrNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	srvURL := srv.URL
	srv.Close()

	dataPath := filepath.Join(t.TempDir(), "links.json")
	c, err := client.New(srvURL, dataPath, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Delete("missing"); !errors.Is(err, client.ErrNotFound) {
		t.Errorf("Delete missing: expected ErrNotFound, got %v", err)
	}

	if _, err := c.Get("missing"); !errors.Is(err, client.ErrNotFound) {
		t.Errorf("Get missing: expected ErrNotFound, got %v", err)
	}
}

func TestFileClientValidationError(t *testing.T) {
	srv, _ := newTestServer(t)
	srvURL := srv.URL
	srv.Close()

	dataPath := filepath.Join(t.TempDir(), "links.json")
	c, err := client.New(srvURL, dataPath, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Set("gh", "not-a-valid-url"); err == nil {
		t.Error("expected validation error for invalid URL")
	}

	if err := c.Set("gh", "ftp://bad-scheme.com"); err == nil {
		t.Error("expected validation error for ftp scheme")
	}
}

func TestBackendSelection(t *testing.T) {
	srv, _ := newTestServer(t)
	dataPath := filepath.Join(t.TempDir(), "links.json")

	// Reachable daemon → HTTPClient
	c, err := client.New(srv.URL, dataPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.(*client.HTTPClient); !ok {
		t.Errorf("expected *client.HTTPClient when daemon is reachable, got %T", c)
	}

	// Unreachable + urlExplicit=false → FileClient
	srvURL := srv.URL
	srv.Close()
	c, err = client.New(srvURL, dataPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.(*client.FileClient); !ok {
		t.Errorf("expected *client.FileClient when daemon is unreachable and urlExplicit=false, got %T", c)
	}

	// Unreachable + urlExplicit=true → error
	_, err = client.New(srvURL, dataPath, true)
	if err == nil {
		t.Error("expected error when daemon is unreachable and urlExplicit=true")
	}
}
