package ipc

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"dragzone/internal/storage"
)

// newDataDir points storage.Dir() at a SHORT temp dir. t.TempDir() bakes the
// (long) test name into the path; combined with the "dragzone.sock" suffix
// that can exceed the macOS AF_UNIX sun_path limit (~104 bytes) for longer
// test names, making net.Listen fail with "bind: invalid argument". A short
// prefix keeps the socket path well under the limit while preserving isolation.
func newDataDir(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "dz")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	t.Setenv(storage.EnvDataDir, dir)
}

func TestServeCallRoundTrip(t *testing.T) {
	newDataDir(t)
	srv, err := Serve(func(req Request) (any, error) {
		if req.Cmd != "echo" {
			return nil, fmt.Errorf("unknown %q", req.Cmd)
		}
		return map[string]any{"args": req.Args, "stack": req.Flags["stack"]}, nil
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	defer srv.Close()

	data, err := Call(Request{Cmd: "echo", Args: []string{"a", "b"}, Flags: map[string]bool{"stack": true}})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	var out struct {
		Args  []string `json:"args"`
		Stack bool     `json:"stack"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Args) != 2 || out.Args[0] != "a" || !out.Stack {
		t.Errorf("round trip = %+v", out)
	}
}

func TestCallPropagatesHandlerError(t *testing.T) {
	newDataDir(t)
	srv, err := Serve(func(Request) (any, error) {
		return nil, fmt.Errorf("boom")
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	if _, err := Call(Request{Cmd: "x"}); err == nil || err.Error() != "boom" {
		t.Errorf("Call error = %v, want boom", err)
	}
}

func TestCallWithoutServerFails(t *testing.T) {
	newDataDir(t)
	if _, err := Call(Request{Cmd: "x"}); err == nil {
		t.Error("Call with no server should fail")
	}
}
