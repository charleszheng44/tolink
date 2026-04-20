package client

import (
	"fmt"
	"net/url"

	"github.com/charleszheng44/tolink/pkg/store"
)

// FileClient wraps pkg/store.Store for direct file access when the daemon is not running.
type FileClient struct {
	s *store.Store
}

func (c *FileClient) List() (map[string]string, error) {
	return c.s.List(), nil
}

func (c *FileClient) Get(shortcut string) (string, error) {
	v, ok := c.s.Get(shortcut)
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

// Set validates the URL scheme before writing, matching the server's validation.
func (c *FileClient) Set(shortcut, rawURL string) error {
	if err := validateURL(rawURL); err != nil {
		return err
	}
	return c.s.Set(shortcut, rawURL)
}

func (c *FileClient) Delete(shortcut string) error {
	return c.s.Delete(shortcut)
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}
