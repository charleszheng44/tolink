package client

import (
	"net/http"
	"time"

	"github.com/charleszheng44/tolink/pkg/store"
)

// ErrNotFound is re-exported from store so callers can use errors.Is(err, client.ErrNotFound).
var ErrNotFound = store.ErrNotFound

// Client manages tolink shortcuts.
type Client interface {
	List() (map[string]string, error)
	Get(shortcut string) (string, error)
	Set(shortcut, url string) error
	Delete(shortcut string) error
}

// New returns an HTTPClient if the daemon at url is reachable (any HTTP response),
// otherwise a FileClient backed by dataPath.
// If urlExplicit is true and the daemon is unreachable, the dial error is returned.
func New(url, dataPath string, urlExplicit bool) (Client, error) {
	hc := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := hc.Get(url + "/.tolink/api/links")
	if err == nil {
		resp.Body.Close()
		return &HTTPClient{base: url}, nil
	}
	if urlExplicit {
		return nil, err
	}
	s, err := store.New(dataPath)
	if err != nil {
		return nil, err
	}
	return &FileClient{s: s}, nil
}
