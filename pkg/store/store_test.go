package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charleszheng44/tolink/pkg/store"
)

func TestNew_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New(filepath.Join(dir, "links.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("expected empty store, got %v", got)
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New(filepath.Join(dir, "links.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Set("gh", "https://github.com"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	url, ok := s.Get("gh")
	if !ok {
		t.Fatal("Get: key not found")
	}
	if url != "https://github.com" {
		t.Fatalf("Get: got %q, want %q", url, "https://github.com")
	}
}

func TestGet_Missing(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.New(filepath.Join(dir, "links.json"))
	if _, ok := s.Get("missing"); ok {
		t.Fatal("expected not found")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.New(filepath.Join(dir, "links.json"))
	s.Set("gh", "https://github.com")
	s.Set("hn", "https://news.ycombinator.com")
	links := s.List()
	if len(links) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(links), links)
	}
	if links["gh"] != "https://github.com" {
		t.Fatalf("unexpected value for gh: %q", links["gh"])
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.New(filepath.Join(dir, "links.json"))
	s.Set("gh", "https://github.com")
	if err := s.Delete("gh"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("gh"); ok {
		t.Fatal("expected key deleted")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.json")
	s, _ := store.New(path)
	s.Set("gh", "https://github.com")

	s2, err := store.New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	url, ok := s2.Get("gh")
	if !ok || url != "https://github.com" {
		t.Fatalf("reload: got %q ok=%v", url, ok)
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.New(filepath.Join(dir, "links.json"))
	s.Set("gh", "https://github.com")

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "links.json" {
		t.Fatalf("expected only links.json, got %v", entries)
	}
}
