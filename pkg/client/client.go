package client

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/charleszheng44/tolink/pkg/store"
)

// ErrNotFound is returned when a shortcut does not exist.
var ErrNotFound = store.ErrNotFound

// Client manages tolink shortcuts.
type Client interface {
	List() (map[string]string, error)
	Get(shortcut string) (string, error)
	Set(shortcut, url string) error
	Delete(shortcut string) error
}

// New returns an HTTPClient if the daemon at baseURL is reachable, otherwise
// a FileClient backed by dataPath. If urlExplicit is true and the daemon is
// unreachable, an error is returned instead of falling back to file mode.
func New(baseURL, dataPath string, urlExplicit bool) (Client, error) {
	dialer := &net.Dialer{Timeout: 500 * time.Millisecond}
	transport := &http.Transport{DialContext: dialer.DialContext}
	probe := &http.Client{
		Transport: transport,
		Timeout:   500 * time.Millisecond,
	}
	resp, err := probe.Get(baseURL + "/.tolink/api/links")
	if err == nil {
		resp.Body.Close()
		return &httpClient{baseURL: baseURL, hc: &http.Client{Timeout: 5 * time.Second}}, nil
	}
	if urlExplicit {
		return nil, fmt.Errorf("cannot connect to %s: %w", baseURL, err)
	}
	return newFileClient(dataPath)
}
