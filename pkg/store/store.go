package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu   sync.RWMutex
	data map[string]string
	path string
}

func New(path string) (*Store, error) {
	s := &Store{path: path, data: make(map[string]string)}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.data)
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tolink-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path)
}

func (s *Store) Get(shortcut string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.data[shortcut]
	return url, ok
}

func (s *Store) List() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

func (s *Store) Set(shortcut, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[shortcut] = url
	return s.save()
}

func (s *Store) Delete(shortcut string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, shortcut)
	return s.save()
}
