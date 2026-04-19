package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mustNew(t *testing.T, path string) *Store {
	t.Helper()
	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New on non-existent file: %v", err)
	}
	if got := s.List(); len(got) != 0 {
		t.Errorf("expected empty store, got %v", got)
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	s := mustNew(t, filepath.Join(dir, "links.json"))
	if err := s.Set("gh", "https://github.com"); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get("gh")
	if !ok {
		t.Fatal("expected key 'gh' to exist")
	}
	if got != "https://github.com" {
		t.Errorf("got %q, want %q", got, "https://github.com")
	}
}

func TestGetMissingKey(t *testing.T) {
	dir := t.TempDir()
	s := mustNew(t, filepath.Join(dir, "links.json"))
	_, ok := s.Get("missing")
	if ok {
		t.Error("expected missing key to return false")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	s := mustNew(t, filepath.Join(dir, "links.json"))
	s.Set("a", "http://a.com")
	s.Set("b", "http://b.com")
	list := s.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
	if list["a"] != "http://a.com" || list["b"] != "http://b.com" {
		t.Errorf("unexpected list contents: %v", list)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	s := mustNew(t, filepath.Join(dir, "links.json"))
	s.Set("gh", "https://github.com")
	if err := s.Delete("gh"); err != nil {
		t.Fatal(err)
	}
	_, ok := s.Get("gh")
	if ok {
		t.Error("expected key 'gh' to be deleted")
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")

	s1 := mustNew(t, path)
	s1.Set("gh", "https://github.com")

	s2, err := New(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	got, ok := s2.Get("gh")
	if !ok {
		t.Fatal("key 'gh' missing after reload")
	}
	if got != "https://github.com" {
		t.Errorf("got %q after reload", got)
	}
}

func TestDeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	s := mustNew(t, filepath.Join(dir, "links.json"))
	err := s.Delete("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAtomicWriteNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	s := mustNew(t, path)
	s.Set("gh", "https://github.com")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "links.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}
}
