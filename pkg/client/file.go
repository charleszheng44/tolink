package client

import "github.com/charleszheng44/tolink/pkg/store"

type fileClient struct {
	s *store.Store
}

func newFileClient(path string) (*fileClient, error) {
	s, err := store.New(path)
	if err != nil {
		return nil, err
	}
	return &fileClient{s: s}, nil
}

func (c *fileClient) List() (map[string]string, error) {
	return c.s.List(), nil
}

func (c *fileClient) Get(shortcut string) (string, error) {
	u, ok := c.s.Get(shortcut)
	if !ok {
		return "", ErrNotFound
	}
	return u, nil
}

func (c *fileClient) Set(shortcut, url string) error {
	return c.s.Set(shortcut, url)
}

func (c *fileClient) Delete(shortcut string) error {
	return c.s.Delete(shortcut)
}
