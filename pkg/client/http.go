package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPClient talks to a running tolink daemon over HTTP.
type HTTPClient struct {
	base string
}

func (c *HTTPClient) List() (map[string]string, error) {
	resp, err := http.Get(c.base + "/.tolink/api/links")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var m map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get filters the full list client-side; the server has no per-shortcut GET endpoint.
func (c *HTTPClient) Get(shortcut string) (string, error) {
	m, err := c.List()
	if err != nil {
		return "", err
	}
	url, ok := m[shortcut]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}

func (c *HTTPClient) Set(shortcut, url string) error {
	body, _ := json.Marshal(struct {
		Shortcut string `json:"shortcut"`
		URL      string `json:"url"`
	}{shortcut, url})
	resp, err := http.Post(c.base+"/.tolink/api/links", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *HTTPClient) Delete(shortcut string) error {
	req, err := http.NewRequest(http.MethodDelete, c.base+"/.tolink/api/links/"+shortcut, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
