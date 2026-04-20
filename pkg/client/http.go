package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type httpClient struct {
	baseURL string
	hc      *http.Client
}

func (c *httpClient) List() (map[string]string, error) {
	resp, err := c.hc.Get(c.baseURL + "/.tolink/api/links")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var links map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return links, nil
}

func (c *httpClient) Get(shortcut string) (string, error) {
	links, err := c.List()
	if err != nil {
		return "", err
	}
	u, ok := links[shortcut]
	if !ok {
		return "", ErrNotFound
	}
	return u, nil
}

func (c *httpClient) Set(shortcut, url string) error {
	body, _ := json.Marshal(struct {
		Shortcut string `json:"shortcut"`
		URL      string `json:"url"`
	}{Shortcut: shortcut, URL: url})
	resp, err := c.hc.Post(
		c.baseURL+"/.tolink/api/links",
		"application/json",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *httpClient) Delete(shortcut string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/.tolink/api/links/"+shortcut, nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
