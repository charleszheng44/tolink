package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store is a thread-safe, file-backed map of shortcut → URL.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
	path string
}

// New loads the store from path, creating the file if it doesn't exist.
func New(path string) (*Store, error) {
	s := &Store{
		data: make(map[string]string),
		path: path,
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&s.data); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return s, nil
}

// Get returns the URL for shortcut and whether it was found.
func (s *Store) Get(shortcut string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[shortcut]
	return v, ok
}

// List returns a copy of all shortcuts.
func (s *Store) List() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

// Set adds or updates a shortcut and persists to disk.
func (s *Store) Set(shortcut, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[shortcut] = url
	return s.save()
}

// Delete removes a shortcut and persists to disk.
func (s *Store) Delete(shortcut string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, shortcut)
	return s.save()
}

// save writes data atomically to s.path. Must be called with mu held.
func (s *Store) save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "links-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
