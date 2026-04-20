package main

import (
	"strings"
	"testing"

	"github.com/charleszheng44/tolink/pkg/client"
)

// fakeClient implements client.Client for testing.
type fakeClient struct {
	links map[string]string
}

func (f *fakeClient) List() (map[string]string, error) {
	out := make(map[string]string, len(f.links))
	for k, v := range f.links {
		out[k] = v
	}
	return out, nil
}

func (f *fakeClient) Get(shortcut string) (string, error) {
	v, ok := f.links[shortcut]
	if !ok {
		return "", client.ErrNotFound
	}
	return v, nil
}

func (f *fakeClient) Set(shortcut, url string) error {
	f.links[shortcut] = url
	return nil
}

func (f *fakeClient) Delete(shortcut string) error {
	if _, ok := f.links[shortcut]; !ok {
		return client.ErrNotFound
	}
	delete(f.links, shortcut)
	return nil
}

func newFake(links map[string]string) *fakeClient {
	m := make(map[string]string, len(links))
	for k, v := range links {
		m[k] = v
	}
	return &fakeClient{links: m}
}

func TestListEmpty(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"list"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("want exit 0, got %d; stderr=%q", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("want empty output, got %q", stdout.String())
	}
}

func TestListOutputSorted(t *testing.T) {
	fc := newFake(map[string]string{
		"zzz": "https://example.com/z",
		"aaa": "https://example.com/a",
		"mmm": "https://example.com/m",
	})
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"list"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("want exit 0, got %d; stderr=%q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d: %q", len(lines), stdout.String())
	}
	if !strings.HasPrefix(lines[0], "aaa") {
		t.Errorf("want first line to start with 'aaa', got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "mmm") {
		t.Errorf("want second line to start with 'mmm', got %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "zzz") {
		t.Errorf("want third line to start with 'zzz', got %q", lines[2])
	}
}

func TestListExtraArgs(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"list", "extra"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d, got %d", exitUsage, code)
	}
}

func TestGetSuccess(t *testing.T) {
	fc := newFake(map[string]string{"gh": "https://github.com"})
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"get", "gh"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("want exit 0, got %d; stderr=%q", code, stderr.String())
	}
	if got := strings.TrimRight(stdout.String(), "\n"); got != "https://github.com" {
		t.Errorf("want %q, got %q", "https://github.com", got)
	}
}

func TestGetNotFound(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"get", "missing"}, &stdout, &stderr)
	if code != exitNotFound {
		t.Errorf("want exit %d, got %d", exitNotFound, code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("want 'not found' in stderr, got %q", stderr.String())
	}
}

func TestGetMissingArg(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"get"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d, got %d", exitUsage, code)
	}
}

func TestAddSuccess(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"add", "gh", "https://github.com"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("want exit 0, got %d; stderr=%q", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("want silent success, got %q", stdout.String())
	}
	if u, err := fc.Get("gh"); err != nil || u != "https://github.com" {
		t.Errorf("shortcut not stored: url=%q err=%v", u, err)
	}
}

func TestAddAlreadyExists(t *testing.T) {
	fc := newFake(map[string]string{"gh": "https://github.com"})
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"add", "gh", "https://other.com"}, &stdout, &stderr)
	if code != exitAlreadyExists {
		t.Errorf("want exit %d, got %d", exitAlreadyExists, code)
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Errorf("want 'already exists' in stderr, got %q", stderr.String())
	}
}

func TestAddBadURL(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"add", "gh", "ftp://github.com"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d (usage), got %d", exitUsage, code)
	}
}

func TestAddMissingArgs(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"add", "gh"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d, got %d", exitUsage, code)
	}
}

func TestUpdateSuccess(t *testing.T) {
	fc := newFake(map[string]string{"gh": "https://github.com"})
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"update", "gh", "https://github.com/new"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("want exit 0, got %d; stderr=%q", code, stderr.String())
	}
	if u, err := fc.Get("gh"); err != nil || u != "https://github.com/new" {
		t.Errorf("shortcut not updated: url=%q err=%v", u, err)
	}
}

func TestUpdateNotFound(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"update", "gh", "https://github.com"}, &stdout, &stderr)
	if code != exitNotFound {
		t.Errorf("want exit %d, got %d", exitNotFound, code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("want 'not found' in stderr, got %q", stderr.String())
	}
}

func TestUpdateBadURL(t *testing.T) {
	fc := newFake(map[string]string{"gh": "https://github.com"})
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"update", "gh", "javascript:alert(1)"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d (usage), got %d", exitUsage, code)
	}
}

func TestUpdateMissingArgs(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"update", "gh"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d, got %d", exitUsage, code)
	}
}

func TestDeleteSuccess(t *testing.T) {
	fc := newFake(map[string]string{"gh": "https://github.com"})
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"delete", "gh"}, &stdout, &stderr)
	if code != exitSuccess {
		t.Fatalf("want exit 0, got %d; stderr=%q", code, stderr.String())
	}
	if _, err := fc.Get("gh"); err == nil {
		t.Error("shortcut should have been deleted")
	}
}

func TestDeleteNotFound(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"delete", "missing"}, &stdout, &stderr)
	if code != exitNotFound {
		t.Errorf("want exit %d, got %d", exitNotFound, code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("want 'not found' in stderr, got %q", stderr.String())
	}
}

func TestDeleteMissingArg(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"delete"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d, got %d", exitUsage, code)
	}
}

func TestUnknownSubcommand(t *testing.T) {
	fc := newFake(nil)
	var stdout, stderr strings.Builder
	code := dispatch(fc, []string{"frobnicate"}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("want exit %d, got %d", exitUsage, code)
	}
}

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		links    map[string]string
		args     []string
		wantCode int
	}{
		{"add existing → 3", map[string]string{"x": "https://x.com"}, []string{"add", "x", "https://y.com"}, exitAlreadyExists},
		{"update missing → 2", nil, []string{"update", "x", "https://x.com"}, exitNotFound},
		{"delete missing → 2", nil, []string{"delete", "x"}, exitNotFound},
		{"get missing → 2", nil, []string{"get", "x"}, exitNotFound},
		{"bad subcommand → 64", nil, []string{"nope"}, exitUsage},
		{"list extra arg → 64", nil, []string{"list", "x"}, exitUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := newFake(tt.links)
			var stdout, stderr strings.Builder
			got := dispatch(fc, tt.args, &stdout, &stderr)
			if got != tt.wantCode {
				t.Errorf("want exit %d, got %d; stderr=%q", tt.wantCode, got, stderr.String())
			}
		})
	}
}
