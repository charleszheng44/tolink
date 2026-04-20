package main

import (
	"bytes"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleszheng44/tolink/pkg/server"
	"github.com/charleszheng44/tolink/pkg/store"
)

var e2eBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "tolinkctl-e2e-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}

	e2eBin = filepath.Join(tmp, "tolinkctl")
	if out, err := exec.Command("go", "build", "-o", e2eBin, ".").CombinedOutput(); err != nil {
		panic("go build: " + string(out) + err.Error())
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func newE2EServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "links.json"))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(server.New(st, nil))
}

func runBin(t *testing.T, serverURL string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(e2eBin, append([]string{"--url", serverURL}, args...)...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestE2ESmoke(t *testing.T) {
	ts := newE2EServer(t)
	defer ts.Close()
	u := ts.URL

	// list: empty store
	stdout, stderr, code := runBin(t, u, "list")
	if code != exitSuccess {
		t.Fatalf("list empty: exit %d; stderr=%q", code, stderr)
	}
	if stdout != "" {
		t.Errorf("list empty: want empty stdout, got %q", stdout)
	}

	// add gh
	_, stderr, code = runBin(t, u, "add", "gh", "https://github.com")
	if code != exitSuccess {
		t.Fatalf("add gh: exit %d; stderr=%q", code, stderr)
	}

	// get gh
	stdout, _, code = runBin(t, u, "get", "gh")
	if code != exitSuccess {
		t.Fatalf("get gh: exit %d", code)
	}
	if got := strings.TrimRight(stdout, "\n"); got != "https://github.com" {
		t.Errorf("get gh: want %q, got %q", "https://github.com", got)
	}

	// add gh again → already exists (exit 3)
	_, stderr, code = runBin(t, u, "add", "gh", "https://other.com")
	if code != exitAlreadyExists {
		t.Errorf("add duplicate: want exit %d, got %d; stderr=%q", exitAlreadyExists, code, stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("add duplicate: want 'already exists' in stderr, got %q", stderr)
	}

	// update gh
	_, stderr, code = runBin(t, u, "update", "gh", "https://github.com/new")
	if code != exitSuccess {
		t.Fatalf("update gh: exit %d; stderr=%q", code, stderr)
	}

	// get gh after update
	stdout, _, code = runBin(t, u, "get", "gh")
	if code != exitSuccess {
		t.Fatalf("get updated gh: exit %d", code)
	}
	if got := strings.TrimRight(stdout, "\n"); got != "https://github.com/new" {
		t.Errorf("get updated gh: want %q, got %q", "https://github.com/new", got)
	}

	// update missing shortcut → exit 2
	_, stderr, code = runBin(t, u, "update", "missing", "https://x.com")
	if code != exitNotFound {
		t.Errorf("update missing: want exit %d, got %d; stderr=%q", exitNotFound, code, stderr)
	}

	// list: should contain gh
	stdout, _, code = runBin(t, u, "list")
	if code != exitSuccess {
		t.Fatalf("list: exit %d", code)
	}
	if !strings.Contains(stdout, "gh") {
		t.Errorf("list: want 'gh' in output, got %q", stdout)
	}

	// delete gh
	_, stderr, code = runBin(t, u, "delete", "gh")
	if code != exitSuccess {
		t.Fatalf("delete gh: exit %d; stderr=%q", code, stderr)
	}

	// get gh after delete → exit 2
	_, stderr, code = runBin(t, u, "get", "gh")
	if code != exitNotFound {
		t.Errorf("get after delete: want exit %d, got %d; stderr=%q", exitNotFound, code, stderr)
	}

	// delete missing → exit 2
	_, stderr, code = runBin(t, u, "delete", "missing")
	if code != exitNotFound {
		t.Errorf("delete missing: want exit %d, got %d; stderr=%q", exitNotFound, code, stderr)
	}
}
