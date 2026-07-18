package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"dragzone/internal/ipc"
	"dragzone/internal/storage"
)

// errFakeTarget is the canned handler error used to exercise the
// {ok:false, error:"..."} response path.
var errFakeTarget = fmt.Errorf("target %q not found", "NoSuchTarget")

// capture runs fn with os.Stdout redirected and returns what it printed.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	defer func() { os.Stdout = orig }()
	os.Stdout = w
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// captureStderr runs fn with os.Stderr redirected and returns what it printed.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	defer func() { os.Stderr = orig }()
	os.Stderr = w
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// newDataDir points storage.Dir() (and therefore the IPC socket path) at a
// SHORT temp dir. t.TempDir() bakes the (long) test name into the path;
// combined with the "dragzone.sock" suffix that can exceed the macOS AF_UNIX
// sun_path limit (~104 bytes) for longer test names, making net.Listen fail
// with "bind: invalid argument". A short prefix keeps the socket path well
// under the limit while preserving isolation between tests.
func newDataDir(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "dz")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	t.Setenv(storage.EnvDataDir, dir)
}

// startFakeServer starts a real ipc.Server backed by handler and arranges
// for it to be closed at test end.
func startFakeServer(t *testing.T, handler ipc.Handler) *ipc.Server {
	t.Helper()
	srv, err := ipc.Serve(handler)
	if err != nil {
		t.Fatalf("ipc.Serve: %v", err)
	}
	t.Cleanup(srv.Close)
	return srv
}

func TestPrintResultListTable(t *testing.T) {
	data := json.RawMessage(`[{"label":"Desktop","action":"folder","events":"dragged"}]`)
	out := capture(t, func() { printResult("list", data, false) })
	if !strings.Contains(out, "Desktop") || !strings.Contains(out, "folder") {
		t.Errorf("list output = %q", out)
	}
}

func TestPrintResultListItems(t *testing.T) {
	data := json.RawMessage(`[{"label":"a.txt","kind":"files","locked":true}]`)
	out := capture(t, func() { printResult("list-items", data, false) })
	if !strings.Contains(out, "1. a.txt (files) [locked]") {
		t.Errorf("list-items output = %q", out)
	}
}

func TestPrintResultJSONPassthrough(t *testing.T) {
	data := json.RawMessage(`{"any":"thing"}`)
	out := capture(t, func() { printResult("list", data, true) })
	if strings.TrimSpace(out) != `{"any":"thing"}` {
		t.Errorf("json passthrough = %q", out)
	}
}

func TestPrintResultStringAndNull(t *testing.T) {
	if out := capture(t, func() { printResult("add", json.RawMessage(`"added 2 item(s)"`), false) }); strings.TrimSpace(out) != "added 2 item(s)" {
		t.Errorf("string result = %q", out)
	}
	if out := capture(t, func() { printResult("clear", json.RawMessage(`null`), false) }); strings.TrimSpace(out) != "" {
		t.Errorf("null result should print nothing, got %q", out)
	}
}

// TestRunSubcommands drives run() for every dz subcommand against a fake IPC
// server, asserting both the JSON request the CLI sent (cmd/args/flags) and
// how it rendered the canned response.
func TestRunSubcommands(t *testing.T) {
	cases := []struct {
		name          string
		args          []string
		canned        any
		wantCmd       string
		wantArgs      []string
		wantFlags     map[string]bool
		wantExit      int
		wantOutSubstr string
	}{
		{
			name:          "list",
			args:          []string{"list"},
			canned:        []map[string]string{{"label": "Desktop", "action": "folder", "events": "dragged"}},
			wantCmd:       "list",
			wantArgs:      []string{},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "Desktop",
		},
		{
			name:          "run",
			args:          []string{"run", "MyAction", "dragged", "nonexistent-file.txt"},
			canned:        "ran MyAction",
			wantCmd:       "run",
			wantArgs:      []string{"MyAction", "dragged", "nonexistent-file.txt"},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "ran MyAction",
		},
		{
			name:          "add",
			args:          []string{"add", "--stack", "nonexistent-file.txt"},
			canned:        "added 1 item(s)",
			wantCmd:       "add",
			wantArgs:      []string{"nonexistent-file.txt"},
			wantFlags:     map[string]bool{"stack": true},
			wantOutSubstr: "added 1 item(s)",
		},
		{
			name:          "list-items",
			args:          []string{"list-items"},
			canned:        []map[string]any{{"label": "a.txt", "kind": "files", "locked": true}},
			wantCmd:       "list-items",
			wantArgs:      []string{},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "1. a.txt (files) [locked]",
		},
		{
			name:          "list-items --json",
			args:          []string{"list-items", "--json"},
			canned:        []map[string]any{{"label": "a.txt", "kind": "files", "locked": false}},
			wantCmd:       "list-items",
			wantArgs:      []string{},
			wantFlags:     map[string]bool{"json": true},
			wantOutSubstr: `"label":"a.txt"`,
		},
		{
			name:          "rename",
			args:          []string{"rename", "1", "new-name"},
			canned:        "renamed",
			wantCmd:       "rename",
			wantArgs:      []string{"1", "new-name"},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "renamed",
		},
		{
			name:          "rename --reset",
			args:          []string{"rename", "1", "--reset"},
			canned:        "reset",
			wantCmd:       "rename",
			wantArgs:      []string{"1"},
			wantFlags:     map[string]bool{"reset": true},
			wantOutSubstr: "reset",
		},
		{
			name:          "remove",
			args:          []string{"remove", "2"},
			canned:        "removed",
			wantCmd:       "remove",
			wantArgs:      []string{"2"},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "removed",
		},
		{
			name:          "lock",
			args:          []string{"lock", "1"},
			canned:        "locked",
			wantCmd:       "lock",
			wantArgs:      []string{"1"},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "locked",
		},
		{
			name:          "unlock",
			args:          []string{"unlock", "1"},
			canned:        "unlocked",
			wantCmd:       "unlock",
			wantArgs:      []string{"1"},
			wantFlags:     map[string]bool{},
			wantOutSubstr: "unlocked",
		},
		{
			name:      "clear",
			args:      []string{"clear"},
			canned:    nil,
			wantCmd:   "clear",
			wantArgs:  []string{},
			wantFlags: map[string]bool{},
		},
		{
			name:      "open",
			args:      []string{"open"},
			canned:    nil,
			wantCmd:   "open",
			wantArgs:  []string{},
			wantFlags: map[string]bool{},
		},
		{
			name:      "close",
			args:      []string{"close"},
			canned:    nil,
			wantCmd:   "close",
			wantArgs:  []string{},
			wantFlags: map[string]bool{},
		},
		{
			name:      "open-dropbar",
			args:      []string{"open-dropbar"},
			canned:    nil,
			wantCmd:   "open-dropbar",
			wantArgs:  []string{},
			wantFlags: map[string]bool{},
		},
		{
			name:      "close-dropbar",
			args:      []string{"close-dropbar"},
			canned:    nil,
			wantCmd:   "close-dropbar",
			wantArgs:  []string{},
			wantFlags: map[string]bool{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newDataDir(t)

			var gotReq ipc.Request
			startFakeServer(t, func(req ipc.Request) (any, error) {
				gotReq = req
				return tc.canned, nil
			})

			var exit int
			out := capture(t, func() { exit = run(tc.args) })

			if exit != 0 {
				t.Errorf("exit = %d, want 0 (stdout=%q)", exit, out)
			}
			if gotReq.Cmd != tc.wantCmd {
				t.Errorf("request cmd = %q, want %q", gotReq.Cmd, tc.wantCmd)
			}
			if len(gotReq.Args) != len(tc.wantArgs) {
				t.Fatalf("request args = %v, want %v", gotReq.Args, tc.wantArgs)
			}
			for i, a := range tc.wantArgs {
				if gotReq.Args[i] != a {
					t.Errorf("request args[%d] = %q, want %q", i, gotReq.Args[i], a)
				}
			}
			for k, v := range tc.wantFlags {
				if gotReq.Flags[k] != v {
					t.Errorf("request flags[%q] = %v, want %v", k, gotReq.Flags[k], v)
				}
			}
			if len(gotReq.Flags) != len(tc.wantFlags) {
				t.Errorf("request flags = %v, want %v", gotReq.Flags, tc.wantFlags)
			}
			if tc.wantOutSubstr != "" && !strings.Contains(out, tc.wantOutSubstr) {
				t.Errorf("output = %q, want substring %q", out, tc.wantOutSubstr)
			}
		})
	}
}

// TestRunAddResolvesExistingFileToAbsolutePath verifies the `add`/`run`
// preprocessing step: a FILE argument that exists on disk is rewritten to
// its absolute path before being sent over the wire, but an argument that
// isn't a real file (an action name, an index, a flag value) passes through
// untouched.
func TestRunAddResolvesExistingFileToAbsolutePath(t *testing.T) {
	newDataDir(t)

	dir := t.TempDir()
	file := dir + "/exists.txt"
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotReq ipc.Request
	startFakeServer(t, func(req ipc.Request) (any, error) {
		gotReq = req
		return "added 1 item(s)", nil
	})

	var exit int
	capture(t, func() { exit = run([]string{"add", file}) })
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if len(gotReq.Args) != 1 || gotReq.Args[0] != file {
		t.Errorf("request args = %v, want [%q] (already absolute, unchanged)", gotReq.Args, file)
	}
}

// TestRunNoArgsPrintsUsage covers the invalid-invocation path: no subcommand
// at all prints usage to stderr and returns exit code 2, without touching
// the IPC socket.
func TestRunNoArgsPrintsUsage(t *testing.T) {
	newDataDir(t) // no server; run must not even attempt to dial it

	var exit int
	errOut := captureStderr(t, func() { exit = run(nil) })

	if exit != 2 {
		t.Errorf("exit = %d, want 2", exit)
	}
	if !strings.Contains(errOut, "usage: dz COMMAND") {
		t.Errorf("stderr = %q, want usage text", errOut)
	}
}

// TestRunConnectionErrorIsReportedCleanly covers the case where no DragZone
// instance is listening on the control socket: run() should print a clean
// "dz: ..." message to stderr and return exit code 1, without panicking.
func TestRunConnectionErrorIsReportedCleanly(t *testing.T) {
	newDataDir(t) // no server started against this socket path

	var exit int
	errOut := captureStderr(t, func() { exit = run([]string{"list"}) })

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	if !strings.HasPrefix(errOut, "dz: ") {
		t.Errorf("stderr = %q, want dz: prefix", errOut)
	}
	if !strings.Contains(errOut, "not running") {
		t.Errorf("stderr = %q, want a not-running message", errOut)
	}
}

// TestRunHandlerErrorIsSurfacedCleanly covers the {ok:false, error:"..."}
// response path: when the app-side handler rejects the request, run() must
// surface the error message on stderr and return exit code 1.
func TestRunHandlerErrorIsSurfacedCleanly(t *testing.T) {
	newDataDir(t)

	startFakeServer(t, func(req ipc.Request) (any, error) {
		return nil, errFakeTarget
	})

	var exit int
	errOut := captureStderr(t, func() { exit = run([]string{"run", "NoSuchTarget", "dragged"}) })

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	want := "dz: " + errFakeTarget.Error()
	if strings.TrimSpace(errOut) != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}
