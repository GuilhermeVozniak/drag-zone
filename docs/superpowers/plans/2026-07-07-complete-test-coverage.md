# Complete Test Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the DragZone desktop app (Go backend + React frontend) to comprehensive test coverage across every functionality area, matching the coverage `apps/web` and `packages/shared` already have.

**Architecture:** Add Go unit/integration tests using the codebase's established patterns (`noopServices` fake, `httptest`, `t.Setenv(storage.EnvDataDir, t.TempDir())`); make the five network actions' endpoints injectable so their round-trips are testable against `httptest`; add the missing React test infrastructure (jsdom + Testing Library + a manual mock of `@/lib/backend`) and test `lib/`, `hooks/`, and the logic-bearing feature components.

**Tech Stack:** Go 1.x + `testing`/`httptest`; Vitest 4 + jsdom + `@testing-library/react` + `@testing-library/user-event` + `@testing-library/jest-dom`; bun workspaces + turbo.

## Global Constraints

- **Spec:** `docs/superpowers/specs/2026-07-07-test-coverage-design.md` — this plan implements it verbatim.
- **Data isolation (MANDATORY):** every Go test that touches a store must call `t.Setenv(storage.EnvDataDir, t.TempDir())` or it writes to the real `~/Library/Application Support/DragZone`.
- **Directory discipline:** `go`/`wails` commands run from `apps/desktop`, never the repo root or `frontend/`. Frontend uses **bun**.
- **Embed dependency:** `go test` embeds `frontend/dist`; run `wails build` on a clean tree before `go test ./...`. Individual `go test ./internal/...` package runs do NOT need the embed (only `package main` does).
- **Native seam is out of scope:** never call real `platform.*` cgo functions that drive AppKit in a test (`SetStatusState`, `ShowGrid`, `SetPinned`, `StartDrag`, `InitNative`, `SetHandlers`, `ClipboardFilePaths`). Test only Go-side pure logic and methods that avoid those.
- **No live third-party I/O:** Imgur/TinyURL/S3/Drive/GitHub round-trips go through `httptest` or injected seams only.
- **Vendored code untouched:** do not test or modify `components/ui/*` (shadcn primitives).
- **Formatting:** `gofmt -l .` must print nothing; `bun run lint` (biome) must be clean.
- **Commit rhythm:** one commit per task, on the `test-coverage` branch. Message form: `test(<area>): <what>`.

## TDD Rhythm (applies to every task)

Each task below lists **Files**, an **Interfaces** block where relevant, the **complete test code**, the **run command with expected output**, and the **commit**. Follow this cycle for each:

1. **Write the test file** exactly as given.
2. **Run it and watch it fail** for the expected reason (missing seam for refactor tasks; for pure-coverage tasks that exercise existing code, the test should *pass* immediately — if it *fails*, you've found a real bug: stop and use superpowers:systematic-debugging, do not "fix" the test to match buggy behavior).
3. For refactor tasks: **make the minimal production change** shown.
4. **Run until green.**
5. **`gofmt -w` / biome** the touched files, then **commit**.

> **Characterization-test note:** Most backend/frontend targets here already have working implementations, so their new tests are *characterization tests* — they pin current correct behavior. "Red first" applies literally only to the refactor tasks (Phase 3, and the s3/cmd-dz extractions), which introduce a new seam the test needs.

---

## Phase 1 — Go pure-logic packages (no production changes)

### Task 1: `internal/model` value semantics

**Files:**
- Test: `apps/desktop/internal/model/model_test.go` (create)

- [ ] Write `apps/desktop/internal/model/model_test.go`:

```go
package model

import "testing"

func TestPayloadHasModifier(t *testing.T) {
	p := Payload{Modifiers: []string{"Option", "Shift"}}
	if !p.HasModifier("Option") {
		t.Error("HasModifier(Option) = false, want true")
	}
	if p.HasModifier("Command") {
		t.Error("HasModifier(Command) = true, want false")
	}
	if (Payload{}).HasModifier("Option") {
		t.Error("empty payload should have no modifiers")
	}
}

func TestPayloadIsEmpty(t *testing.T) {
	cases := []struct {
		name string
		p    Payload
		want bool
	}{
		{"nothing", Payload{}, true},
		{"paths", Payload{Paths: []string{"/a"}}, false},
		{"text", Payload{Text: "hi"}, false},
		{"empty slice", Payload{Paths: []string{}}, true},
	}
	for _, c := range cases {
		if got := c.p.IsEmpty(); got != c.want {
			t.Errorf("%s: IsEmpty() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTargetOption(t *testing.T) {
	t1 := Target{Options: map[string]string{"mode": "copy", "blank": ""}}
	if got := t1.Option("mode", "move"); got != "copy" {
		t.Errorf(`Option("mode") = %q, want "copy"`, got)
	}
	// An empty stored value falls through to the default.
	if got := t1.Option("blank", "fallback"); got != "fallback" {
		t.Errorf(`Option("blank") = %q, want "fallback"`, got)
	}
	if got := t1.Option("missing", "def"); got != "def" {
		t.Errorf(`Option("missing") = %q, want "def"`, got)
	}
	// Nil map must not panic.
	if got := (Target{}).Option("x", "d"); got != "d" {
		t.Errorf("nil-map Option = %q, want d", got)
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/model/ -v` — Expected: PASS (3 tests).
- [ ] Commit: `test(model): cover Payload and Target value semantics`

### Task 2: `internal/storage` round-trip and error paths

**Files:**
- Test: `apps/desktop/internal/storage/storage_test.go` (create)

- [ ] Write `apps/desktop/internal/storage/storage_test.go`:

```go
package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	want := sample{Name: "a", Count: 3}
	if err := Save("s.json", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var got sample
	if err := Load("s.json", &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

func TestLoadMissingFileLeavesValueUntouched(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	pre := sample{Name: "default", Count: 1}
	got := pre
	if err := Load("nope.json", &got); err != nil {
		t.Fatalf("Load of missing file must not error: %v", err)
	}
	if got != pre {
		t.Errorf("missing-file Load mutated value to %+v", got)
	}
}

func TestLoadMalformedJSONErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got sample
	if err := Load("bad.json", &got); err == nil {
		t.Error("Load of malformed JSON should error")
	}
}

func TestSaveIsAtomicAndPretty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	if err := Save("p.json", sample{Name: "x", Count: 2}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "p.json"))
	if err != nil {
		t.Fatal(err)
	}
	// storage.Save writes json.MarshalIndent output verbatim (no trailing
	// newline), so assert pretty-printing via the 2-space indent instead.
	if !filepath.IsAbs(dir) || len(b) == 0 || !strings.Contains(string(b), "\n  ") {
		t.Errorf("unexpected file content: %q", b)
	}
	// No leftover temp files from the atomic write.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected exactly p.json, got %d entries", len(entries))
	}
}

func TestDirHonorsEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	got, err := Dir()
	if err != nil || got != dir {
		t.Errorf("Dir() = %q, %v; want %q", got, err, dir)
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/storage/ -v` — Expected: PASS (5 tests).
- [ ] Commit: `test(storage): cover save/load round-trip, missing and malformed files`

### Task 3: `internal/config` defaults, scale, persistence

**Files:**
- Test: `apps/desktop/internal/config/config_test.go` (create)

- [ ] Write `apps/desktop/internal/config/config_test.go`:

```go
package config

import (
	"math"
	"testing"

	"dragzone/internal/storage"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.GlobalShortcut != "F3" || d.PopOutShortcut != "F4" {
		t.Errorf("shortcut defaults wrong: %+v", d)
	}
	if d.GridColumns != 4 || d.GridSize != 33 || d.Theme != "system" {
		t.Errorf("grid defaults wrong: %+v", d)
	}
	if !d.AnimateGrid || !d.ShowKeyOverlays || !d.PlaySounds || !d.DragOverlay ||
		!d.NotifyOnComplete || !d.AutoUpdateCheck {
		t.Errorf("boolean defaults wrong: %+v", d)
	}
}

func TestScaleClamp(t *testing.T) {
	cases := []struct {
		grid int
		want float64
	}{
		{0, 0.8}, {100, 1.4}, {33, 0.8 + 33.0/100*0.6},
		{-50, 0.8}, {250, 1.4}, // clamped
	}
	for _, c := range cases {
		got := Settings{GridSize: c.grid}.Scale()
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Scale(gridSize=%d) = %v, want %v", c.grid, got, c.want)
		}
	}
}

func TestLoadDefaultsThenSetPersists(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	// Fresh load with no file returns defaults.
	st, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Get().GridColumns != 4 {
		t.Errorf("fresh Load did not return defaults: %+v", st.Get())
	}
	// Set persists; a fresh Load sees the change.
	s := st.Get()
	s.GridColumns = 6
	s.Theme = "dark"
	if err := st.Set(s); err != nil {
		t.Fatalf("Set: %v", err)
	}
	st2, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if st2.Get().GridColumns != 6 || st2.Get().Theme != "dark" {
		t.Errorf("persisted settings = %+v", st2.Get())
	}
	// Unset fields keep their default (merge over Defaults()).
	if st2.Get().GlobalShortcut != "F3" {
		t.Errorf("defaults not preserved on reload: %+v", st2.Get())
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/config/ -v` — Expected: PASS (3 tests).
- [ ] Commit: `test(config): cover defaults, scale clamp, load/set persistence`

### Task 4: `internal/ipc` socket round-trip

**Files:**
- Test: `apps/desktop/internal/ipc/ipc_test.go` (create)

**Interfaces:**
- Consumes: `Serve(Handler) (*Server, error)`, `Call(Request) (json.RawMessage, error)`, `SocketPath()`. `Serve`/`Call` resolve the socket under `storage.Dir()`, so `t.Setenv(storage.EnvDataDir, t.TempDir())` points them at a temp socket.

- [ ] Write `apps/desktop/internal/ipc/ipc_test.go`:

```go
package ipc

import (
	"encoding/json"
	"fmt"
	"testing"

	"dragzone/internal/storage"
)

func TestServeCallRoundTrip(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
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
	t.Setenv(storage.EnvDataDir, t.TempDir())
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
	t.Setenv(storage.EnvDataDir, t.TempDir())
	if _, err := Call(Request{Cmd: "x"}); err == nil {
		t.Error("Call with no server should fail")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/ipc/ -v` — Expected: PASS (3 tests).
- [ ] Commit: `test(ipc): cover control-socket serve/call round-trip and errors`

### Task 5: `internal/actions` registry

**Files:**
- Test: `apps/desktop/internal/actions/registry_test.go` (create)

**Interfaces:**
- Consumes: `NewRegistry()`, `(*Registry).Register`, `.TryRegister`, `.Get`, `.Specs`. Needs a tiny fake `Action` — `Spec() model.ActionSpec` only.

- [ ] Write `apps/desktop/internal/actions/registry_test.go`:

```go
package actions

import (
	"testing"

	"dragzone/internal/model"
)

type fakeAction struct{ id string }

func (f fakeAction) Spec() model.ActionSpec { return model.ActionSpec{ID: f.id, Name: f.id} }

func TestRegistryRegisterGetSpecsOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeAction{"a"})
	r.Register(fakeAction{"b"})

	if _, err := r.Get("a"); err != nil {
		t.Errorf("Get(a): %v", err)
	}
	if _, err := r.Get("missing"); err == nil {
		t.Error("Get(missing) should error")
	}
	specs := r.Specs()
	if len(specs) != 2 || specs[0].ID != "a" || specs[1].ID != "b" {
		t.Errorf("Specs order = %+v, want [a b]", specs)
	}
}

func TestSpecsNeverNil(t *testing.T) {
	if specs := NewRegistry().Specs(); specs == nil {
		t.Error("Specs() on empty registry returned nil")
	}
}

func TestTryRegisterRejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.TryRegister(fakeAction{"dup"}); err != nil {
		t.Fatal(err)
	}
	if err := r.TryRegister(fakeAction{"dup"}); err == nil {
		t.Error("duplicate TryRegister should error")
	}
}

func TestRegisterPanicsOnDuplicate(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeAction{"x"})
	defer func() {
		if recover() == nil {
			t.Error("Register of a duplicate should panic")
		}
	}()
	r.Register(fakeAction{"x"})
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/ -v` — Expected: PASS (4 tests).
- [ ] Commit: `test(actions): cover registry register/get/specs/duplicate`

---

## Phase 2 — Built-in actions, no production change

All Phase 2 & 3 test files are `package builtin`. The helpers `nullProgress` (in `zip_test.go`) and `mustWrite` (in `folder_test.go`) already exist in the package and are reused — do NOT redeclare them. Add a shared recording `fakeServices` in a new `builtin_test_helpers_test.go` so multiple files can use it (the `bundles` package has its own copy; the `builtin` package needs its own).

### Task 6: shared builtin test helper

**Files:**
- Test: `apps/desktop/internal/actions/builtin/builtin_test_helpers_test.go` (create)

**Interfaces:**
- Produces: `recServices` — a recording `actions.Services` with fields `Clipboard string`, `Trashed [][]string`, `AirDropped [][]string`, `Notes []string`, `Sounds []string`, `Opened []string`, `ClipboardErr error`, `ReadClip string`. Later tasks construct `&recServices{}` and assert on these fields.

- [ ] Write `apps/desktop/internal/actions/builtin/builtin_test_helpers_test.go`:

```go
package builtin

// recServices is a recording actions.Services fake shared across builtin tests.
type recServices struct {
	Clipboard    string
	ClipboardErr error
	Trashed      [][]string
	AirDropped   [][]string
	Notes        []string
	Sounds       []string
	Opened       []string
	ReadClip     string
}

func (r *recServices) CopyToClipboard(s string) error {
	if r.ClipboardErr != nil {
		return r.ClipboardErr
	}
	r.Clipboard = s
	return nil
}
func (r *recServices) ReadClipboard() (string, error) { return r.ReadClip, nil }
func (r *recServices) Notify(title, body string)      { r.Notes = append(r.Notes, title+"|"+body) }
func (r *recServices) PlaySound(name string)          { r.Sounds = append(r.Sounds, name) }
func (r *recServices) OpenURL(u string) error         { r.Opened = append(r.Opened, u); return nil }
func (r *recServices) OpenPath(p string) error        { r.Opened = append(r.Opened, p); return nil }
func (r *recServices) Reveal(p string) error          { r.Opened = append(r.Opened, p); return nil }
func (r *recServices) Trash(p []string) error         { r.Trashed = append(r.Trashed, p); return nil }
func (r *recServices) AirDrop(p []string) error       { r.AirDropped = append(r.AirDropped, p); return nil }
```

- [ ] Run: `cd apps/desktop && go vet ./internal/actions/builtin/` — Expected: no output (compiles). Commit Tasks 6 and 7 together so `recServices` is never transiently unused.
- [ ] Commit (with Task 7): `test(builtin): add shared recording Services fake`

### Task 7: clipboard, trash, airdrop (Services-backed)

**Files:**
- Test: `apps/desktop/internal/actions/builtin/simple_actions_test.go` (create)

- [ ] Write `apps/desktop/internal/actions/builtin/simple_actions_test.go`:

```go
package builtin

import (
	"context"
	"errors"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func inv(p model.Payload, svc actions.Services) actions.Invocation {
	return actions.Invocation{
		Target:   model.Target{Label: "t"},
		Payload:  p,
		Progress: nullProgress{},
		Services: svc,
	}
}

func TestClipboardCopiesTextAndPaths(t *testing.T) {
	svc := &recServices{}
	res, err := CopyToClipboard{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemText, Text: "hello"}, svc))
	if err != nil || svc.Clipboard != "hello" || res.Message != "Copied to clipboard" {
		t.Fatalf("text: clip=%q res=%+v err=%v", svc.Clipboard, res, err)
	}
	svc = &recServices{}
	_, err = CopyToClipboard{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b"}}, svc))
	if err != nil || svc.Clipboard != "/a\n/b" {
		t.Fatalf("files: clip=%q err=%v", svc.Clipboard, err)
	}
}

func TestClipboardSurfacesServiceError(t *testing.T) {
	svc := &recServices{ClipboardErr: errors.New("no clip")}
	if _, err := CopyToClipboard{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemText, Text: "x"}, svc)); err == nil {
		t.Error("clipboard error should propagate")
	}
}

func TestTrashDelegatesToService(t *testing.T) {
	svc := &recServices{}
	res, err := Trash{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b"}}, svc))
	if err != nil || len(svc.Trashed) != 1 || len(svc.Trashed[0]) != 2 {
		t.Fatalf("trashed=%v err=%v", svc.Trashed, err)
	}
	if res.Message != "Moved 2 item(s) to Trash" {
		t.Errorf("message = %q", res.Message)
	}
}

func TestAirDropDelegatesToService(t *testing.T) {
	svc := &recServices{}
	res, err := AirDrop{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}}, svc))
	if err != nil || len(svc.AirDropped) != 1 || res.Message != "Sharing 1 item(s) via AirDrop" {
		t.Fatalf("airdropped=%v res=%+v err=%v", svc.AirDropped, res, err)
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run 'Clipboard|Trash|AirDrop' -v` — Expected: PASS (4 tests).
- [ ] Commit: `test(builtin): cover clipboard, trash, airdrop actions`

### Task 8: save-text (snippetName + Dropped)

**Files:**
- Test: `apps/desktop/internal/actions/builtin/savetext_test.go` (create)

- [ ] Write `apps/desktop/internal/actions/builtin/savetext_test.go`:

```go
package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestSnippetName(t *testing.T) {
	if got := snippetName("Hello there world"); got != "Hello there world.txt" {
		t.Errorf("snippetName = %q", got)
	}
	// Filesystem-unsafe characters are stripped.
	if got := snippetName(`a/b:c*d`); strings.ContainsAny(got, `/\:*?"<>|`) {
		t.Errorf("snippetName kept unsafe chars: %q", got)
	}
	// Whitespace-only text yields a timestamped fallback name.
	if got := snippetName("   "); !strings.HasPrefix(got, "Snippet ") || !strings.HasSuffix(got, ".txt") {
		t.Errorf("fallback name = %q", got)
	}
	// Capped at six words.
	if got := snippetName("one two three four five six seven eight"); len(strings.Fields(strings.TrimSuffix(got, ".txt"))) > 6 {
		t.Errorf("snippetName not capped: %q", got)
	}
}

func TestSaveTextWritesFile(t *testing.T) {
	dir := t.TempDir()
	res, err := SaveText{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"path": dir}},
		Payload:  model.Payload{Kind: model.ItemText, Text: "Meeting notes here"},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Message, "Saved ") {
		t.Errorf("message = %q", res.Message)
	}
	b, err := os.ReadFile(filepath.Join(dir, "Meeting notes here.txt"))
	if err != nil || string(b) != "Meeting notes here" {
		t.Errorf("file content = %q err %v", b, err)
	}
}

func TestSaveTextRejectsEmptyConfigOrText(t *testing.T) {
	if _, err := SaveText{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemText, Text: "x"},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("missing folder should error")
	}
	if _, err := SaveText{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"path": t.TempDir()}},
		Payload:  model.Payload{Kind: model.ItemText, Text: "   "},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("blank text should error")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run 'SaveText|SnippetName' -v` — Expected: PASS.
- [ ] Commit: `test(builtin): cover save-text naming and file writing`

### Task 9: open-app + install-app pure helpers

**Files:**
- Test: `apps/desktop/internal/actions/builtin/apps_test.go` (create)

- [ ] Write `apps/desktop/internal/actions/builtin/apps_test.go`:

```go
package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppName(t *testing.T) {
	if got := appName("/Applications/Safari.app"); got != "Safari" {
		t.Errorf("appName = %q", got)
	}
	if got := appName("Notes"); got != "Notes" {
		t.Errorf("appName without .app = %q", got)
	}
}

func TestMountPointFromPlist(t *testing.T) {
	plist := `<plist><dict><key>system-entities</key><array><dict>
	<key>mount-point</key>
	<string>/Volumes/My App</string>
	</dict></array></dict></plist>`
	if got := mountPointFromPlist(plist); got != "/Volumes/My App" {
		t.Errorf("mountPointFromPlist = %q", got)
	}
	if got := mountPointFromPlist("<plist></plist>"); got != "" {
		t.Errorf("no mount point should be empty, got %q", got)
	}
}

func TestFindApp(t *testing.T) {
	// Top-level .app
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Cool.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := findApp(dir); err != nil || filepath.Base(got) != "Cool.app" {
		t.Errorf("findApp top-level = %q err %v", got, err)
	}
	// Nested one level deep
	nest := t.TempDir()
	sub := filepath.Join(nest, "folder")
	if err := os.MkdirAll(filepath.Join(sub, "Deep.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := findApp(nest); err != nil || filepath.Base(got) != "Deep.app" {
		t.Errorf("findApp nested = %q err %v", got, err)
	}
	// None
	if _, err := findApp(t.TempDir()); err == nil {
		t.Error("findApp with no .app should error")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run 'AppName|MountPoint|FindApp' -v` — Expected: PASS.
- [ ] Commit: `test(builtin): cover open-app and install-app pure helpers`

### Task 10: FTP `collectUploadEntries` traversal

**Files:**
- Test: `apps/desktop/internal/actions/builtin/ftp_test.go` (create)

- [ ] Write `apps/desktop/internal/actions/builtin/ftp_test.go`:

```go
package builtin

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCollectUploadEntriesFlattensDirs(t *testing.T) {
	root := t.TempDir()
	// A single loose file plus a directory tree.
	loose := filepath.Join(root, "loose.txt")
	if err := os.WriteFile(loose, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(filepath.Join(tree, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "a.txt"), []byte("ab"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "sub", "b.txt"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, total, err := collectUploadEntries([]string{loose, tree})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5+2+1 {
		t.Errorf("total bytes = %d, want 8", total)
	}
	var rels []string
	for _, e := range entries {
		rels = append(rels, e.rel)
	}
	sort.Strings(rels)
	want := []string{"loose.txt", "tree/a.txt", "tree/sub/b.txt"}
	if len(rels) != 3 || rels[0] != want[0] || rels[1] != want[1] || rels[2] != want[2] {
		t.Errorf("rels = %v, want %v", rels, want)
	}
}

func TestCollectUploadEntriesErrors(t *testing.T) {
	if _, _, err := collectUploadEntries([]string{"/does/not/exist/xyz"}); err == nil {
		t.Error("missing path should error")
	}
	if _, _, err := collectUploadEntries([]string{t.TempDir()}); err == nil {
		t.Error("empty dir (no files) should error with 'nothing to upload'")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run CollectUploadEntries -v` — Expected: PASS.
- [ ] Commit: `test(builtin): cover ftp/s3 upload-entry traversal`

---

## Phase 3 — Network actions: injectable endpoints (refactor + test)

Each task here makes a **minimal** production change — replace a hardcoded endpoint constant/literal with an unexported package-level `var` defaulting to the production URL — then tests the round-trip against `httptest`. This is the one place "red first" applies: the test references the new `var`, so it fails to compile until the seam exists.

### Task 11: Imgur — injectable endpoint + upload round-trip

**Files:**
- Modify: `apps/desktop/internal/actions/builtin/imgur.go`
- Test: `apps/desktop/internal/actions/builtin/imgur_test.go` (create)

**Interfaces:**
- Produces: package var `imgurAPIURL string` (default `"https://api.imgur.com/3/image"`), consumed by `uploadToImgur`.

- [ ] Write the test `apps/desktop/internal/actions/builtin/imgur_test.go`:

```go
package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestIsImageFile(t *testing.T) {
	for _, ok := range []string{"a.jpg", "a.JPEG", "b.png", "c.gif", "d.webp", "e.heic", "f.bmp", "g.tiff"} {
		if !isImageFile(ok) {
			t.Errorf("isImageFile(%q) = false", ok)
		}
	}
	for _, no := range []string{"a.txt", "b.pdf", "noext"} {
		if isImageFile(no) {
			t.Errorf("isImageFile(%q) = true", no)
		}
	}
}

func TestImgurUploadRoundTrip(t *testing.T) {
	var gotAuth, gotContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		_ = r.ParseMultipartForm(1 << 20)
		w.Write([]byte(`{"data":{"link":"https://i.imgur.com/abc.png"}}`))
	}))
	defer ts.Close()
	old := imgurAPIURL
	imgurAPIURL = ts.URL
	defer func() { imgurAPIURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(img, []byte("\x89PNG\r\n\x1a\nfake"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &recServices{}
	res, err := ImgurUpload{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"client_id": "CID"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if res.URL != "https://i.imgur.com/abc.png" || svc.Clipboard != res.URL {
		t.Errorf("url=%q clip=%q", res.URL, svc.Clipboard)
	}
	if gotAuth != "Client-ID CID" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if len(gotContentType) < 9 || gotContentType[:9] != "multipart" {
		t.Errorf("content-type = %q", gotContentType)
	}
}

func TestImgurRejectsNonImageAndMissingID(t *testing.T) {
	svc := &recServices{}
	// missing client id
	if _, err := ImgurUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{}, Payload: model.Payload{Paths: []string{"/a.png"}},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("missing client_id should error")
	}
	// non-image file
	txt := filepath.Join(t.TempDir(), "a.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	if _, err := ImgurUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{"client_id": "C"}},
		Payload:  model.Payload{Paths: []string{txt}},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("non-image should error")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run Imgur` — Expected: FAIL to compile (`undefined: imgurAPIURL`).
- [ ] Make the production change in `imgur.go`: add near the top, after the imports:

```go
// imgurAPIURL is the upload endpoint; a package var so tests can point it at
// an httptest server.
var imgurAPIURL = "https://api.imgur.com/3/image"
```

  and in `uploadToImgur`, change the request URL literal to `imgurAPIURL`:

```go
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imgurAPIURL, pr)
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run 'Imgur|IsImageFile' -v` — Expected: PASS.
- [ ] `gofmt -w internal/actions/builtin/imgur.go` then commit: `test(builtin): make Imgur endpoint injectable and cover upload`

### Task 12: Shorten URL — injectable endpoint + round-trip

**Files:**
- Modify: `apps/desktop/internal/actions/builtin/shorten.go`
- Test: `apps/desktop/internal/actions/builtin/shorten_test.go` (create)

**Interfaces:**
- Produces: package var `tinyURLAPI string` (default `"https://tinyurl.com/api-create.php"`), consumed by `shortenWithTinyURL` which appends `?url=`.

- [ ] Write `apps/desktop/internal/actions/builtin/shorten_test.go`:

```go
package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestParseHTTPURL(t *testing.T) {
	if _, err := parseHTTPURL("  https://a.com/x  "); err != nil {
		t.Errorf("valid url errored: %v", err)
	}
	for _, bad := range []string{"", "ftp://a.com", "notaurl", "http://"} {
		if _, err := parseHTTPURL(bad); err == nil {
			t.Errorf("parseHTTPURL(%q) should error", bad)
		}
	}
}

func TestShortenRoundTripAndClickReadsClipboard(t *testing.T) {
	var gotURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Query().Get("url")
		w.Write([]byte("https://tinyurl.com/abc"))
	}))
	defer ts.Close()
	old := tinyURLAPI
	tinyURLAPI = ts.URL
	defer func() { tinyURLAPI = old }()

	// Dropped path
	svc := &recServices{}
	res, err := ShortenURL{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com/very/long"},
		Progress: nullProgress{}, Services: svc,
	})
	if err != nil || res.URL != "https://tinyurl.com/abc" || svc.Clipboard != res.URL {
		t.Fatalf("dropped: res=%+v clip=%q err=%v", res, svc.Clipboard, err)
	}
	if gotURL != "https://example.com/very/long" {
		t.Errorf("forwarded url = %q", gotURL)
	}

	// Clicked path reads the clipboard
	svc2 := &recServices{ReadClip: "https://example.com/from-clip"}
	if _, err := ShortenURL{}.Clicked(context.Background(), actions.Invocation{
		Progress: nullProgress{}, Services: svc2,
	}); err != nil {
		t.Fatalf("clicked: %v", err)
	}
	if gotURL != "https://example.com/from-clip" {
		t.Errorf("clicked forwarded url = %q", gotURL)
	}
}

func TestShortenServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := tinyURLAPI
	tinyURLAPI = ts.URL
	defer func() { tinyURLAPI = old }()
	if _, err := ShortenURL{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://a.com"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("500 from tinyurl should error")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run Shorten` — Expected: FAIL to compile (`undefined: tinyURLAPI`).
- [ ] Production change in `shorten.go`: add the package var and use it in `shortenWithTinyURL`:

```go
// tinyURLAPI is the TinyURL create endpoint; a package var for test injection.
var tinyURLAPI = "https://tinyurl.com/api-create.php"
```

  and change the request line to:

```go
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tinyURLAPI+"?url="+url.QueryEscape(longURL), nil)
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run 'Shorten|ParseHTTPURL' -v` — Expected: PASS.
- [ ] `gofmt -w` then commit: `test(builtin): make Shorten endpoint injectable and cover round-trip`

### Task 13: Add-ons list — injectable endpoint + parse

**Files:**
- Modify: `apps/desktop/internal/addons/addons.go`
- Test: `apps/desktop/internal/addons/addons_test.go` (create)

**Interfaces:**
- Produces: change `contentsURL` from `const` to `var` (keep the same value). Consumed by `List`.

- [ ] Write `apps/desktop/internal/addons/addons_test.go`:

```go
package addons

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFiltersDzbundleDirs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header: %q", r.Header.Get("Accept"))
		}
		w.Write([]byte(`[
			{"name":"Alpha.dzbundle","type":"dir"},
			{"name":"Beta.dzbundle","type":"dir"},
			{"name":"README.md","type":"file"},
			{"name":"NotABundle","type":"dir"}
		]`))
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()

	names, err := List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Beta" {
		t.Errorf("names = %v, want [Alpha Beta]", names)
	}
}

func TestListServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()
	if _, err := List(context.Background()); err == nil {
		t.Error("403 should error")
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/addons/ -run List` — Expected: FAIL to compile (`cannot assign to contentsURL`).
- [ ] Production change in `addons.go`: split the `const (...)` block so `contentsURL` becomes a `var`:

```go
// contentsURL is a var so tests can point List at an httptest server.
var contentsURL = "https://api.github.com/repos/aptonic/dropzone4-actions/contents/"

const (
	archiveURL  = "https://codeload.github.com/aptonic/dropzone4-actions/zip/refs/heads/master"
	cacheMaxAge = 24 * time.Hour
)
```

- [ ] Run: `cd apps/desktop && go test ./internal/addons/ -v` — Expected: PASS.
- [ ] `gofmt -w` then commit: `test(addons): make contents endpoint injectable and cover List`

### Task 14: S3 public-URL formatting (extract helper) + gdrive pure helpers

**Files:**
- Modify: `apps/desktop/internal/actions/builtin/s3.go`
- Modify: `apps/desktop/internal/actions/builtin/gdrive.go`
- Test: `apps/desktop/internal/actions/builtin/s3_test.go` (create)
- Test: `apps/desktop/internal/actions/builtin/gdrive_test.go` (create)

**Interfaces:**
- Produces: `s3PublicURL(bucket, region, urlPrefix, key string) string` in `s3.go`, extracted from the inline logic in `S3Upload.Dropped`. `driveFilesURL` becomes a `var` in `gdrive.go`.

- [ ] Write `apps/desktop/internal/actions/builtin/s3_test.go`:

```go
package builtin

import "testing"

func TestS3PublicURL(t *testing.T) {
	// Default virtual-hosted-style URL.
	got := s3PublicURL("mybucket", "us-east-1", "", "uploads/a.png")
	want := "https://mybucket.s3.us-east-1.amazonaws.com/uploads/a.png"
	if got != want {
		t.Errorf("default = %q, want %q", got, want)
	}
	// Custom prefix wins and trailing slash is trimmed.
	got = s3PublicURL("b", "eu-west-1", "https://cdn.example.com/", "k/x.jpg")
	if got != "https://cdn.example.com/k/x.jpg" {
		t.Errorf("custom prefix = %q", got)
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run S3PublicURL` — Expected: FAIL to compile (`undefined: s3PublicURL`).
- [ ] Production change in `s3.go`: extract the helper and call it in `Dropped`. Add:

```go
// s3PublicURL builds the URL copied to the clipboard: the custom prefix when
// set, otherwise the default virtual-hosted-style S3 URL.
func s3PublicURL(bucket, region, urlPrefix, key string) string {
	if urlPrefix != "" {
		return strings.TrimRight(urlPrefix, "/") + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
}
```

  and replace the inline block in `Dropped` (the `resultURL := fmt.Sprintf(...)` line plus the following `if prefix := ...` block) with:

```go
	resultURL := s3PublicURL(bucket, region, inv.Target.Option("url_prefix", ""), firstKey)
```

- [ ] Write `apps/desktop/internal/actions/builtin/gdrive_test.go`:

```go
package builtin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRandomURLTokenUniqueAndURLSafe(t *testing.T) {
	a, err := randomURLToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := randomURLToken()
	if a == "" || a == b {
		t.Errorf("tokens not unique/non-empty: %q %q", a, b)
	}
	if strings.ContainsAny(a, "+/=") {
		t.Errorf("token not URL-safe: %q", a)
	}
}

func TestDriveCheckResponse(t *testing.T) {
	ok := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}
	if err := driveCheckResponse(ok); err != nil {
		t.Errorf("200 should be nil: %v", err)
	}
	unauth := &http.Response{StatusCode: 401, Status: "401 Unauthorized", Body: io.NopCloser(strings.NewReader("bad token"))}
	err := driveCheckResponse(unauth)
	if err == nil || !isDriveAuthError(err) {
		t.Errorf("401 should be a drive auth error, got %v", err)
	}
	serverErr := &http.Response{StatusCode: 500, Status: "500 err", Body: io.NopCloser(strings.NewReader("x"))}
	if err := driveCheckResponse(serverErr); err == nil || isDriveAuthError(err) {
		t.Errorf("500 should be a non-auth error, got %v", err)
	}
}

func TestDriveWebViewLink(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "webViewLink") {
			t.Errorf("missing fields query: %q", r.URL.RawQuery)
		}
		w.Write([]byte(`{"webViewLink":"https://drive.google.com/file/d/ID/view"}`))
	}))
	defer ts.Close()
	old := driveFilesURL
	driveFilesURL = ts.URL + "/"
	defer func() { driveFilesURL = old }()

	link, err := driveWebViewLink(context.Background(), ts.Client(), "ID")
	if err != nil || link != "https://drive.google.com/file/d/ID/view" {
		t.Errorf("link = %q err %v", link, err)
	}
}
```

- [ ] Production change in `gdrive.go`: move `driveFilesURL` out of the `const (...)` block into its own `var driveFilesURL = "https://www.googleapis.com/drive/v3/files/"` (keep the value). Leave the other Drive endpoints as consts.
- [ ] Run: `cd apps/desktop && go test ./internal/actions/builtin/ -run 'S3PublicURL|Drive|RandomURLToken' -v` — Expected: PASS.
- [ ] `gofmt -w` the two files, then commit: `test(builtin): cover S3 URL formatting and Google Drive helpers`

---

## Phase 4 — Bundles, task runner, and the App facade

### Task 15: `bundles` template + meta edge cases

**Files:**
- Test: `apps/desktop/internal/bundles/template_test.go` (create)
- Test: `apps/desktop/internal/bundles/meta_more_test.go` (create)

`bundles_test.go` already covers `ParseMeta` happy path + `ScriptAction`. Add template + meta error/option coverage. `optionFields` is unexported in `package bundles`, so the meta test lives in that package.

- [ ] Write `apps/desktop/internal/bundles/template_test.go`:

```go
package bundles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTemplateRubyAndPython(t *testing.T) {
	dir := t.TempDir()
	rb, err := CreateTemplate(dir, "My Action", "ruby")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(rb) != "My Action.dzbundle" {
		t.Errorf("bundle path = %q", rb)
	}
	script, err := os.ReadFile(filepath.Join(rb, "action.rb"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "# Name: My Action") || !strings.Contains(string(script), "def dragged") {
		t.Errorf("ruby template missing header/handler")
	}

	py, err := CreateTemplate(dir, "Py Action", "python")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(py, "action.py")); err != nil {
		t.Errorf("python template not written: %v", err)
	}
}

func TestCreateTemplateRejectsBlankAndDuplicate(t *testing.T) {
	dir := t.TempDir()
	if _, err := CreateTemplate(dir, "   ", "ruby"); err == nil {
		t.Error("blank name should error")
	}
	if _, err := CreateTemplate(dir, "Dup", "ruby"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTemplate(dir, "Dup", "ruby"); err == nil {
		t.Error("duplicate name should error")
	}
}
```

- [ ] Write `apps/desktop/internal/bundles/meta_more_test.go`:

```go
package bundles

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "action.py")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseMetaMissingHeaderOrName(t *testing.T) {
	if _, err := ParseMeta(writeScript(t, "print('hi')\n")); err == nil {
		t.Error("missing header should error")
	}
	noName := "# Dropzone Action Info\n# Description: x\n"
	if _, err := ParseMeta(writeScript(t, noName)); err == nil {
		t.Error("missing Name should error")
	}
}

func TestParseMetaDefaultsHandlesAndEvents(t *testing.T) {
	body := "# Dropzone Action Info\n# Name: Minimal\n\nprint('x')\n"
	meta, err := ParseMeta(writeScript(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Events) != 2 || meta.Handles[0] != "Files" {
		t.Errorf("defaults not applied: %+v", meta)
	}
}

func TestOptionFieldsByNIB(t *testing.T) {
	if f := optionFields("Login"); len(f) != 2 || f[0].Key != "username" || f[1].Type != "password" {
		t.Errorf("Login fields = %+v", f)
	}
	if f := optionFields("APIKey"); len(f) != 1 || f[0].Key != "api_key" {
		t.Errorf("APIKey fields = %+v", f)
	}
	if f := optionFields("Unknown"); f != nil {
		t.Errorf("unknown NIB should yield nil, got %+v", f)
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/bundles/ -run 'CreateTemplate|ParseMeta|OptionFields' -v` — Expected: PASS.
- [ ] Commit: `test(bundles): cover template creation and meta parsing edges`

### Task 16: `tasks.Runner` lifecycle

**Files:**
- Test: `apps/desktop/internal/tasks/runner_test.go` (create)

**Interfaces:**
- Consumes: `NewRunner(Config)`, `(*Runner).Run(ctx, act, target, payload, event) (string, error)`, `.List()`, `.Cancel(id)`, `.Dismiss(id)`. Needs a fake `actions.Action` implementing `Dropper`/`Clicker`, and a channel-synced fake `actions.Services`.

**Design:** the runner executes actions in a goroutine, so tests must synchronize. Use a fake action whose `Dropped` blocks on a channel the test controls, and observe task state via `List()` after signaling completion; and a fake `Emit` that pushes onto a buffered channel so the test can wait for the terminal publish.

- [ ] Write `apps/desktop/internal/tasks/runner_test.go`:

```go
package tasks

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// fakeAction runs fn on Dropped/Clicked; both events supported.
type fakeAction struct {
	fn func(actions.Invocation) (actions.Result, error)
}

func (fakeAction) Spec() model.ActionSpec { return model.ActionSpec{ID: "fake", Name: "Fake"} }
func (a fakeAction) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	return a.fn(inv)
}
func (a fakeAction) Clicked(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	return a.fn(inv)
}

type recSvc struct {
	mu     sync.Mutex
	notes  []string
	sounds []string
}

func (s *recSvc) CopyToClipboard(string) error   { return nil }
func (s *recSvc) ReadClipboard() (string, error) { return "", nil }
func (s *recSvc) Notify(t, b string)             { s.mu.Lock(); s.notes = append(s.notes, t); s.mu.Unlock() }
func (s *recSvc) PlaySound(n string)             { s.mu.Lock(); s.sounds = append(s.sounds, n); s.mu.Unlock() }
func (s *recSvc) OpenURL(string) error           { return nil }
func (s *recSvc) OpenPath(string) error          { return nil }
func (s *recSvc) Reveal(string) error            { return nil }
func (s *recSvc) Trash([]string) error           { return nil }
func (s *recSvc) AirDrop([]string) error         { return nil }

// newRunner returns a runner whose Emit signals `changed` on every publish, so
// tests can wait for the terminal state.
func newRunner(t *testing.T, svc actions.Services) (*Runner, chan struct{}) {
	t.Helper()
	changed := make(chan struct{}, 64)
	r := NewRunner(Config{
		Emit:          func(string, ...any) { changed <- struct{}{} },
		Services:      svc,
		NotifyEnabled: func() bool { return true },
		SoundsEnabled: func() bool { return true },
	})
	return r, changed
}

// waitDone polls List() until the single task reaches a terminal status.
func waitDone(t *testing.T, r *Runner) model.TaskState {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		list := r.List()
		if len(list) == 1 && list[0].Status != model.TaskRunning {
			return list[0]
		}
		select {
		case <-deadline:
			t.Fatalf("task did not finish; list=%+v", list)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestRunSuccessNotifiesAndPlaysGlass(t *testing.T) {
	svc := &recSvc{}
	r, _ := newRunner(t, svc)
	act := fakeAction{fn: func(inv actions.Invocation) (actions.Result, error) {
		inv.Progress.Percent(50)
		return actions.Result{Message: "done", URL: "https://x/y"}, nil
	}}
	id, err := r.Run(context.Background(), act, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged)
	if err != nil || id == "" {
		t.Fatalf("Run: id=%q err=%v", id, err)
	}
	st := waitDone(t, r)
	if st.Status != model.TaskDone || st.Detail != "done" || st.Percent != 100 || st.ResultURL != "https://x/y" {
		t.Errorf("final state = %+v", st)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.notes) != 1 || len(svc.sounds) != 1 || svc.sounds[0] != "Glass" {
		t.Errorf("notes=%v sounds=%v", svc.notes, svc.sounds)
	}
}

func TestRunErrorNotifiesAndPlaysBasso(t *testing.T) {
	svc := &recSvc{}
	r, _ := newRunner(t, svc)
	act := fakeAction{fn: func(actions.Invocation) (actions.Result, error) {
		return actions.Result{}, errors.New("kaboom")
	}}
	if _, err := r.Run(context.Background(), act, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged); err != nil {
		t.Fatal(err)
	}
	st := waitDone(t, r)
	if st.Status != model.TaskError || st.Error != "kaboom" {
		t.Errorf("final state = %+v", st)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.sounds) != 1 || svc.sounds[0] != "Basso" {
		t.Errorf("sounds = %v, want [Basso]", svc.sounds)
	}
}

func TestRunRejectsUnsupportedEvent(t *testing.T) {
	r, _ := newRunner(t, &recSvc{})
	// A Dropper-only action cannot be clicked.
	dropOnly := struct {
		actions.Action
		actions.Dropper
	}{}
	_ = dropOnly
	// Use fakeAction but call an invalid event string.
	if _, err := r.Run(context.Background(), fakeAction{fn: func(actions.Invocation) (actions.Result, error) {
		return actions.Result{}, nil
	}}, model.Target{Label: "T"}, model.Payload{}, "bogus"); err == nil {
		t.Error("unknown event should error")
	}
}

func TestDismissRemovesFinishedTask(t *testing.T) {
	r, _ := newRunner(t, &recSvc{})
	id, _ := r.Run(context.Background(), fakeAction{fn: func(actions.Invocation) (actions.Result, error) {
		return actions.Result{Message: "ok"}, nil
	}}, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged)
	waitDone(t, r)
	r.Dismiss(id)
	if len(r.List()) != 0 {
		t.Errorf("dismiss left tasks: %+v", r.List())
	}
}

func TestCancelAbortsRunningTask(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	r, _ := newRunner(t, &recSvc{})
	act := fakeAction{fn: func(inv actions.Invocation) (actions.Result, error) {
		close(started)
		<-release // block until cancelled context fires? we simply wait
		return actions.Result{Message: "late"}, nil
	}}
	id, _ := r.Run(context.Background(), act, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged)
	<-started
	r.Cancel(id)
	close(release)
	st := waitDone(t, r)
	if st.Status != model.TaskError || st.Error != "cancelled" {
		t.Errorf("cancelled task state = %+v", st)
	}
}
```

- [ ] Run: `cd apps/desktop && go test ./internal/tasks/ -v -race` — Expected: PASS (5 tests, no race).
- [ ] Commit: `test(tasks): cover runner success/error/cancel/dismiss lifecycle`

> **Note:** `TestRunRejectsUnsupportedEvent`'s `dropOnly` scaffold is illustrative; the actual assertion uses the `"bogus"` event string, which `Run` rejects before dispatch. Delete the unused `dropOnly` block if biome/govet complains — it's only there to document intent.

### Task 17: App facade — recent shares + saveTargetOption

**Files:**
- Test: `apps/desktop/app_shares_test.go` (create)

**Interfaces:**
- Consumes: `newTestApp(t)` and `noopServices` (already in `app_grid_test.go`, same `package main`). Do NOT redeclare them.

**Design:** `emit` is safe pre-startup (`a.ctx == nil` → no-op), and `addRecentShare`/`RecentShares`/`ClearRecentShares`/`saveTargetOption` touch no cgo, so they test directly. Avoid `taskFeedback` (calls `platform.SetStatusState`).

- [ ] Write `apps/desktop/app_shares_test.go`:

```go
package main

import "testing"

func TestRecentSharesCapAndOrder(t *testing.T) {
	app := newTestApp(t)
	for i := 0; i < 15; i++ {
		app.addRecentShare("title", "https://x/"+string(rune('a'+i)))
	}
	shares := app.RecentShares()
	if len(shares) != 10 {
		t.Fatalf("recent shares capped at 10, got %d", len(shares))
	}
	// Newest first: the last added URL leads.
	if shares[0].URL != "https://x/o" {
		t.Errorf("newest-first order wrong: %q", shares[0].URL)
	}
}

func TestRecentSharesEmptyIsNonNil(t *testing.T) {
	app := newTestApp(t)
	if got := app.RecentShares(); got == nil || len(got) != 0 {
		t.Errorf("empty RecentShares = %v, want []", got)
	}
}

func TestClearRecentSharesPersists(t *testing.T) {
	app := newTestApp(t)
	app.addRecentShare("t", "https://x/1")
	if err := app.ClearRecentShares(); err != nil {
		t.Fatal(err)
	}
	if len(app.RecentShares()) != 0 {
		t.Error("shares not cleared")
	}
}

func TestSaveTargetOptionSetAndDelete(t *testing.T) {
	app := newTestApp(t)
	tgt, err := app.AddTarget("folder", "F", map[string]string{"path": "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	app.saveTargetOption(tgt.ID, "token", "abc")
	got, _ := app.grid.Get(tgt.ID)
	if got.Options["token"] != "abc" {
		t.Errorf("option not saved: %+v", got.Options)
	}
	// Empty value deletes the key.
	app.saveTargetOption(tgt.ID, "token", "")
	got, _ = app.grid.Get(tgt.ID)
	if _, ok := got.Options["token"]; ok {
		t.Errorf("empty value should delete key: %+v", got.Options)
	}
}
```

- [ ] Run: `cd apps/desktop && wails build && go test . -run 'RecentShares|ClearRecentShares|SaveTargetOption' -v` — Expected: PASS. (`wails build` first because `package main` embeds `frontend/dist`.)
- [ ] Commit: `test(app): cover recent-shares cap/order and saveTargetOption`

### Task 18: App facade — Drop Bar bindings

**Files:**
- Test: `apps/desktop/app_dropbar_test.go` (create)

**Design:** Drop Bar bindings mutate `a.dropBar` and `emit` (safe). `DropBarConsume` honors `Locked` and the `DropBarKeepsItems` setting. Avoid `SetDropBarPopOut`/`StartDragOut`/`DropBarPaste` (they call `platform.*`).

- [ ] Write `apps/desktop/app_dropbar_test.go`:

```go
package main

import (
	"testing"

	"dragzone/internal/model"
)

func TestDropBarAddRemoveClear(t *testing.T) {
	app := newTestApp(t)
	it, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 1 {
		t.Fatalf("add: %+v", app.DropBarItems())
	}
	if err := app.DropBarRemove(it.ID); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 0 {
		t.Error("remove failed")
	}
	app.DropBarAdd(model.Payload{Kind: model.ItemText, Text: "hi"})
	if err := app.DropBarClear(); err != nil || len(app.DropBarItems()) != 0 {
		t.Errorf("clear failed: %v", err)
	}
}

func TestDropBarConsumeHonorsLockAndSetting(t *testing.T) {
	app := newTestApp(t)

	// Locked item survives consume.
	locked, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}})
	if err := app.DropBarSetLocked(locked.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := app.DropBarConsume(locked.ID); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 1 {
		t.Error("locked item should survive consume")
	}

	// Unlocked item is removed on consume.
	free, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/b"}})
	if err := app.DropBarConsume(free.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.dropBar.Get(free.ID); ok {
		t.Error("unlocked item should be consumed")
	}

	// With the keep setting on, even unlocked items survive.
	s := app.settings.Get()
	s.DropBarKeepsItems = true
	app.settings.Set(s)
	kept, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/c"}})
	app.DropBarConsume(kept.ID)
	if _, ok := app.dropBar.Get(kept.ID); !ok {
		t.Error("keep-items setting should preserve consumed item")
	}
}

func TestDropBarSeparateAndCombine(t *testing.T) {
	app := newTestApp(t)
	stack, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b", "/c"}})
	if err := app.DropBarSeparate(stack.ID); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 3 {
		t.Fatalf("separate: %+v", app.DropBarItems())
	}
	if err := app.DropBarCombineAll(); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 1 {
		t.Errorf("combine: %+v", app.DropBarItems())
	}
}
```

- [ ] Run: `cd apps/desktop && go test . -run DropBar -v` — Expected: PASS. (Reuses the `wails build` output from Task 17; rebuild only if `frontend/dist` was cleaned.)
- [ ] Commit: `test(app): cover Drop Bar add/remove/consume/separate bindings`

### Task 19: App facade — IPC command dispatch

**Files:**
- Test: `apps/desktop/app_ipc_test.go` (create)

**Design:** `handleIPC` dispatches CLI commands. Test the data-only commands (`list`, `add`, `list-items`, `rename`, `remove`, `lock`/`unlock`, `clear`, unknown) and `ipcRun` error paths (bad event, unknown target). Avoid `open`/`close`/`open-dropbar`/`close-dropbar` (they call `platform.*`).

- [ ] Write `apps/desktop/app_ipc_test.go`:

```go
package main

import (
	"testing"

	"dragzone/internal/ipc"
	"dragzone/internal/model"
)

func TestHandleIPCListAndAdd(t *testing.T) {
	app := newTestApp(t)
	// The default grid seeds 6 targets.
	if rows, err := app.handleIPC(ipc.Request{Cmd: "list"}); err != nil {
		t.Fatalf("list: %v", err)
	} else if rs, ok := rows.([]struct {
		Label  string `json:"label"`
		Action string `json:"action"`
		Events string `json:"events"`
	}); ok && len(rs) == 0 {
		t.Error("list returned no rows")
	}

	// add two files individually.
	if _, err := app.handleIPC(ipc.Request{Cmd: "add", Args: []string{"/x/a.txt", "/x/b.txt"}}); err != nil {
		t.Fatal(err)
	}
	if len(app.dropBar.List()) != 2 {
		t.Errorf("add: %d items", len(app.dropBar.List()))
	}
	// add --stack keeps them as one item.
	app.DropBarClear()
	if _, err := app.handleIPC(ipc.Request{Cmd: "add", Args: []string{"/x/a", "/x/b"}, Flags: map[string]bool{"stack": true}}); err != nil {
		t.Fatal(err)
	}
	if len(app.dropBar.List()) != 1 {
		t.Errorf("add --stack: %d items", len(app.dropBar.List()))
	}
}

func TestHandleIPCItemCommandsByIndex(t *testing.T) {
	app := newTestApp(t)
	app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/x/a.txt"}})
	// rename item 1
	if _, err := app.handleIPC(ipc.Request{Cmd: "rename", Args: []string{"1", "custom"}}); err != nil {
		t.Fatal(err)
	}
	if app.dropBar.List()[0].Label != "custom" {
		t.Errorf("rename failed: %+v", app.dropBar.List()[0])
	}
	// lock / unlock
	if _, err := app.handleIPC(ipc.Request{Cmd: "lock", Args: []string{"1"}}); err != nil {
		t.Fatal(err)
	}
	if !app.dropBar.List()[0].Locked {
		t.Error("lock failed")
	}
	// bad index
	if _, err := app.handleIPC(ipc.Request{Cmd: "remove", Args: []string{"99"}}); err == nil {
		t.Error("out-of-range index should error")
	}
}

func TestHandleIPCUnknownAndRunErrors(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.handleIPC(ipc.Request{Cmd: "frobnicate"}); err == nil {
		t.Error("unknown command should error")
	}
	// run with a bad event
	if _, err := app.handleIPC(ipc.Request{Cmd: "run", Args: []string{"Desktop", "sideways"}}); err == nil {
		t.Error("bad event should error")
	}
	// run with an unknown target label
	if _, err := app.handleIPC(ipc.Request{Cmd: "run", Args: []string{"NoSuchTarget", "dragged"}}); err == nil {
		t.Error("unknown target should error")
	}
}
```

- [ ] Run: `cd apps/desktop && go test . -run HandleIPC -v` — Expected: PASS.
- [ ] Commit: `test(app): cover dz IPC command dispatch and errors`

### Task 20: `cmd/dz` output formatting

**Files:**
- Test: `apps/desktop/cmd/dz/main_test.go` (create)

**Design:** `printResult` writes to stdout; capture it by swapping `os.Stdout` with a pipe. This covers the `list`/`list-items`/JSON/string/null formatting branches without a running app.

- [ ] Write `apps/desktop/cmd/dz/main_test.go`:

```go
package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// capture runs fn with os.Stdout redirected and returns what it printed.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)
	return string(out)
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
```

- [ ] Run: `cd apps/desktop && go test ./cmd/dz/ -v` — Expected: PASS.
- [ ] Commit: `test(dz): cover CLI output formatting branches`

---

## Phase 5 — Frontend test infrastructure

The desktop frontend currently has **no** test runner wiring beyond the bare
`"test": "vitest run"` script. This phase stands up jsdom + Testing Library and
the two seams every later test depends on:

1. **A manual mock of `@/lib/backend`** — the single module that re-exports the
   generated Wails bindings. Component/hook behaviour tests `vi.mock("@/lib/backend")`
   so they never touch the (gitignored, generated) `wailsjs/*` tree.
2. **Regex `test.alias` stubs** for the two modules that import `wailsjs/*`
   *directly* rather than through the backend facade — `lib/backend.ts`
   (`wailsjs/go/main/App`, `wailsjs/runtime/runtime`) and `lib/dnd.ts`
   (`wailsjs/runtime/runtime`). Tests that exercise the *real* `backend.ts`
   (the `uiScale` unit) or the *real* `dnd.ts` need these so the module graph
   resolves on a clean tree where `wailsjs/` does not exist.

> Why both mechanisms? `vi.mock` replaces the whole `@/lib/backend` module for
> behaviour tests. But `uiScale` lives *in* `backend.ts` and must be tested
> against the real implementation — importing it pulls in the real `wailsjs`
> imports, which only the `test.alias` stubs satisfy. `dnd.ts` likewise imports
> `OnFileDrop` straight from the runtime and is tested for real.

> The `wailsjs/go/models` import in `backend.ts` is `import type` only and is
> erased by esbuild at transpile — no stub required.

### Task 21 — Stand up jsdom + Testing Library + backend mock

No red-first step here: this is infra plus a smoke test that passes on arrival.

- [ ] Add dev dependencies to `apps/desktop/frontend/package.json` (`devDependencies` block):

```jsonc
"@testing-library/dom": "^10.4.0",
"@testing-library/jest-dom": "^6.6.3",
"@testing-library/react": "^16.3.0",
"@testing-library/user-event": "^14.6.1",
"jsdom": "^26.0.0",
```

- [ ] Run: `cd apps/desktop/frontend && bun install` — Expected: lockfile updates, deps resolve.

- [ ] Replace `apps/desktop/frontend/vite.config.ts` with a version that adds a
  `test` block. Keep the existing `@` alias and plugins untouched; add jsdom,
  globals, the setup file, and the `wailsjs` → stub aliases (test-only so the
  production build is unaffected):

```ts
/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    // Stub the generated Wails modules that a few source files import
    // directly. Test-only: never applied to `vite build`.
    // NOTE: the leading `.*` in each `find` is REQUIRED — rollup-alias (behind
    // Vite's resolve.alias) resolves a regex alias as `id.replace(find, repl)`,
    // so without it the `../../` prefix of the specifier survives and the
    // rewritten id (`../..//abs/stub.ts`) fails to resolve.
    alias: [
      {
        find: /.*wailsjs\/runtime\/runtime$/,
        replacement: path.resolve(__dirname, './src/test/stubs/runtime.ts'),
      },
      {
        find: /.*wailsjs\/go\/main\/App$/,
        replacement: path.resolve(__dirname, './src/test/stubs/App.ts'),
      },
    ],
  },
})
```

- [ ] Create `apps/desktop/frontend/src/test/setup.ts` — registers the jest-dom
  matchers, polyfills the DOM APIs Radix primitives touch at render time (jsdom
  ships none of them), and auto-cleans the DOM between tests:

```ts
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Radix UI primitives (Slider/Select/Dialog/Switch) call these at render or
// interaction time; jsdom implements none of them.
if (!globalThis.ResizeObserver) {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver
}
Element.prototype.scrollIntoView ||= () => {}
Element.prototype.hasPointerCapture ||= () => false
Element.prototype.setPointerCapture ||= () => {}
Element.prototype.releasePointerCapture ||= () => {}

afterEach(() => {
  cleanup()
})
```

- [ ] Create `apps/desktop/frontend/src/test/stubs/runtime.ts` — a stand-in for
  `wailsjs/runtime/runtime`. `OnFileDrop` records its callback so tests can
  fire synthetic native drops; the exported `__*` helpers are test-only:

```ts
import { vi } from 'vitest'

type FileDropCb = (x: number, y: number, paths: string[]) => void
let fileDropCb: FileDropCb | null = null

export const OnFileDrop = vi.fn((cb: FileDropCb, _useDropTarget: boolean) => {
  fileDropCb = cb
})
export const EventsOn = vi.fn((_event: string, _cb: (...a: unknown[]) => void) => () => {})
export const EventsEmit = vi.fn()

// --- test helpers (not part of the real runtime API) ---
export function __emitFileDrop(x: number, y: number, paths: string[]) {
  fileDropCb?.(x, y, paths)
}
export function __resetRuntimeStub() {
  fileDropCb = null
  OnFileDrop.mockClear()
  EventsOn.mockClear()
  EventsEmit.mockClear()
}
```

- [ ] Create `apps/desktop/frontend/src/test/stubs/App.ts` — the real
  `backend.ts` reads `App.GetSettings` etc. at module-eval to build its object
  literal; an empty namespace leaves those fields `undefined`, which is fine for
  the `uiScale`-only test that pulls in the real module:

```ts
// Empty stand-in for the generated wailsjs App bindings. The real backend.ts
// only *references* these at module load; the uiScale test never calls them.
export {}
```

- [ ] Create the manual mock `apps/desktop/frontend/src/lib/__mocks__/backend.ts`.
  It mirrors the shape of the real `backend`/`events` exports, exposes an event
  registry so tests can fire backend events, and re-implements `uiScale` for the
  rare consumer that imports it through a mocked module:

```ts
import { vi } from 'vitest'

// Every binding is an async no-op by default; tests override with
// mockResolvedValue where they assert on a return.
const afn = () => vi.fn(async () => undefined)

export const backend = {
  settings: { get: afn(), set: afn() },
  actions: { specs: afn(), installBundle: afn(), openFolder: afn(), develop: afn() },
  grid: {
    list: afn(), add: afn(), addFromPaths: afn(), update: afn(),
    duplicate: afn(), remove: afn(), move: afn(),
  },
  drop: afn(),
  click: afn(),
  tasks: { list: afn(), dismiss: afn(), cancel: afn() },
  shares: { list: afn(), clear: afn(), open: afn() },
  playDropSound: afn(),
  dropBar: {
    list: afn(), add: afn(), remove: afn(), clear: afn(), consume: afn(),
    setLocked: afn(), rename: afn(), setPopOut: afn(), separate: afn(),
    combineAll: afn(), copyToClipboard: afn(), reveal: afn(), paste: afn(),
  },
  quickLook: afn(),
  answerInput: afn(),
  addons: { list: afn(), install: afn() },
  cli: { installed: afn(), install: afn() },
  updates: { check: afn(), version: afn() },
  dialogs: { chooseFolder: afn(), chooseApplication: afn() },
  dragOut: afn(),
  fileIcon: afn(),
  openURL: afn(),
  window: { hide: afn(), quit: afn(), about: afn() },
}

// --- event registry: each subscriber records its latest callback + unsub ---
export const __eventCbs: Record<string, ((...a: unknown[]) => void) | null> = {}
export const __unsub: Record<string, ReturnType<typeof vi.fn>> = {}

function sub(name: string) {
  return vi.fn((fn: (...a: unknown[]) => void) => {
    __eventCbs[name] = fn
    const u = vi.fn()
    __unsub[name] = u
    return u
  })
}

export const events = {
  onGridChanged: sub('grid:changed'),
  onTasksChanged: sub('tasks:changed'),
  onDropBarChanged: sub('dropbar:changed'),
  onOpenSettings: sub('settings:open'),
  onSpecsChanged: sub('specs:changed'),
  onDropBarPopOut: sub('dropbar:popout'),
  onInputRequest: sub('input:request'),
  onWindowVisibility: sub('window:visibility'),
  onWindowBeak: sub('window:beak'),
  onSharesChanged: sub('shares:changed'),
}

// --- test helpers ---
export function __fireEvent(name: string, ...args: unknown[]) {
  __eventCbs[name]?.(...args)
}
export function __resetBackendMock() {
  for (const k of Object.keys(__eventCbs)) __eventCbs[k] = null
}

// Real formula (mirrors config.Settings.Scale) so mocked consumers still work.
export function uiScale(s: { gridSize?: number } | null): number {
  const pct = Math.min(100, Math.max(0, s?.gridSize ?? 33))
  return 0.8 + (pct / 100) * 0.6
}
```

> Note the two-module trick used by later tests: import the live values
> (`backend`, `events`) from `@/lib/backend` (the mocked module) and the
> test-only helpers (`__fireEvent`, `__unsub`, `__resetBackendMock`) from
> `@/lib/__mocks__/backend` directly. Both specifiers resolve to the *same*
> file, so vitest returns the same module instance — the helpers see the exact
> callbacks the hooks registered. Importing helpers from the real path also
> keeps TypeScript happy (the real `@/lib/backend` has no `__*` exports).

- [ ] Exclude test files, stubs, and the manual mock from the production
  `tsc` build so `bun run build` stays green (they use vitest globals /
  jest-dom matchers / test-only exports that the app build must not typecheck).
  Edit `apps/desktop/frontend/tsconfig.json` to add an `exclude`:

```jsonc
"include": ["src", "wailsjs"],
"exclude": [
  "src/**/*.test.ts",
  "src/**/*.test.tsx",
  "src/test",
  "src/lib/__mocks__"
],
```

- [ ] Write `apps/desktop/frontend/src/test/smoke.test.tsx` — proves jsdom +
  plugin-react JSX + jest-dom matchers all wire up:

```tsx
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

describe('test infrastructure', () => {
  it('renders JSX into jsdom and matches with jest-dom', () => {
    render(<div>hello dragzone</div>)
    expect(screen.getByText('hello dragzone')).toBeInTheDocument()
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: the smoke test PASSES.
- [ ] Run: `cd apps/desktop/frontend && bunx tsc --noEmit` — Expected: no errors (build tsconfig ignores test files).
- [ ] Commit: `test(frontend): stand up jsdom + testing-library + backend mock`

---

## Phase 6 — Frontend lib + hooks

Pure-logic units in `lib/` and the event/data-loading hooks in `hooks/`. Each
task is characterization (behaviour already correct → tests pass on arrival).

### Task 22 — `lib/utils` (`cn`) + `lib/backend` (`uiScale`)

- [ ] Write `apps/desktop/frontend/src/lib/utils.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { cn } from '@/lib/utils'

describe('cn', () => {
  it('joins truthy class names', () => {
    expect(cn('a', 'b')).toBe('a b')
  })
  it('drops falsy values', () => {
    expect(cn('a', false, null, undefined, 'b')).toBe('a b')
  })
  it('merges conflicting tailwind classes so the last wins', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4')
  })
  it('supports the conditional-object form', () => {
    expect(cn('base', { active: true, hidden: false })).toBe('base active')
  })
})
```

- [ ] Write `apps/desktop/frontend/src/lib/backend.test.ts` — exercises the
  *real* `uiScale` (resolved through the `test.alias` wailsjs stubs):

```ts
import { describe, expect, it } from 'vitest'
import { uiScale, type Settings } from '@/lib/backend'

const settings = (gridSize: number): Settings => ({ gridSize }) as Settings

describe('uiScale', () => {
  it('defaults to the 33% grid size when settings are null', () => {
    expect(uiScale(null)).toBeCloseTo(0.998, 3) // 0.8 + 0.33*0.6
  })
  it('maps gridSize 0 to the minimum scale 0.8', () => {
    expect(uiScale(settings(0))).toBeCloseTo(0.8, 5)
  })
  it('maps gridSize 100 to the maximum scale 1.4', () => {
    expect(uiScale(settings(100))).toBeCloseTo(1.4, 5)
  })
  it('clamps gridSize above 100', () => {
    expect(uiScale(settings(150))).toBeCloseTo(1.4, 5)
  })
  it('clamps negative gridSize up to 0.8', () => {
    expect(uiScale(settings(-20))).toBeCloseTo(0.8, 5)
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover cn + uiScale helpers`

### Task 23 — `lib/dnd` (payload extraction + native drop routing)

- [ ] Write `apps/desktop/frontend/src/lib/dnd.test.ts`:

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { initNativeFileDrop, payloadFromDataTransfer, setUIScale } from '@/lib/dnd'
import { __emitFileDrop, __resetRuntimeStub } from '@/test/stubs/runtime'

function fakeDT(data: Record<string, string>): DataTransfer {
  return { getData: (t: string) => data[t] ?? '' } as unknown as DataTransfer
}

beforeEach(() => {
  __resetRuntimeStub()
  setUIScale(1)
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('payloadFromDataTransfer', () => {
  it('reads a url from the first non-comment uri-list line, trimmed', () => {
    const dt = fakeDT({ 'text/uri-list': '# comment\nhttps://example.com  \n' })
    expect(payloadFromDataTransfer(dt)).toEqual({ kind: 'url', text: 'https://example.com' })
  })
  it('prefers a url over plain text when both are present', () => {
    const dt = fakeDT({ 'text/uri-list': 'https://a.test', 'text/plain': 'ignored' })
    expect(payloadFromDataTransfer(dt)).toEqual({ kind: 'url', text: 'https://a.test' })
  })
  it('falls back to plain text when there is no uri-list', () => {
    const dt = fakeDT({ 'text/plain': 'hello world' })
    expect(payloadFromDataTransfer(dt)).toEqual({ kind: 'text', text: 'hello world' })
  })
  it('returns null when the transfer carries neither url nor text', () => {
    expect(payloadFromDataTransfer(fakeDT({}))).toBeNull()
  })
  it('returns null when the uri-list holds only comment lines', () => {
    // no non-comment uri and no text/plain -> null
    expect(payloadFromDataTransfer(fakeDT({ 'text/uri-list': '# only a comment' }))).toBeNull()
  })
})

describe('initNativeFileDrop', () => {
  it('resolves the drop-id from the element under the cursor', () => {
    document.body.innerHTML = `<div data-drop-id="dropbar"><span id="child"></span></div>`
    vi.spyOn(document, 'elementFromPoint').mockReturnValue(document.getElementById('child'))
    const onFiles = vi.fn()
    initNativeFileDrop({ onFiles })
    __emitFileDrop(100, 200, ['/a.txt'])
    expect(onFiles).toHaveBeenCalledWith('dropbar', ['/a.txt'])
  })
  it('un-zooms the cursor coordinates by the UI scale before hit-testing', () => {
    setUIScale(2)
    const spy = vi.spyOn(document, 'elementFromPoint').mockReturnValue(null)
    initNativeFileDrop({ onFiles: vi.fn() })
    __emitFileDrop(100, 200, ['/a'])
    expect(spy).toHaveBeenCalledWith(50, 100)
  })
  it('passes a null drop-id when nothing is under the cursor', () => {
    vi.spyOn(document, 'elementFromPoint').mockReturnValue(null)
    const onFiles = vi.fn()
    initNativeFileDrop({ onFiles })
    __emitFileDrop(1, 1, ['/a'])
    expect(onFiles).toHaveBeenCalledWith(null, ['/a'])
  })
  it('clears the native-dragging body class on drop', () => {
    document.body.classList.add('native-dragging')
    vi.spyOn(document, 'elementFromPoint').mockReturnValue(null)
    initNativeFileDrop({ onFiles: vi.fn() })
    __emitFileDrop(1, 1, [])
    expect(document.body.classList.contains('native-dragging')).toBe(false)
  })
})
```

> jsdom does no layout, so `document.elementFromPoint` always returns null; we
> `vi.spyOn` it to return the element we want under the (un-zoomed) cursor.

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover dnd payload extraction + native drop routing`

### Task 24 — `lib/icons` (`iconFor` + `tileStyleFor`)

- [ ] Write `apps/desktop/frontend/src/lib/icons.test.ts`:

```ts
import { Archive, File, Wifi } from 'lucide-react'
import { describe, expect, it } from 'vitest'
import { iconFor, tileStyleFor } from '@/lib/icons'

describe('iconFor', () => {
  it('returns the mapped icon for a known name', () => {
    expect(iconFor('archive')).toBe(Archive)
    expect(iconFor('wifi')).toBe(Wifi)
  })
  it('falls back to File for an unknown name', () => {
    expect(iconFor('does-not-exist')).toBe(File)
  })
})

describe('tileStyleFor', () => {
  it('returns the branded style for a known action id', () => {
    const s = tileStyleFor('airdrop', 'wifi')
    expect(s.glyph).toBe(Wifi)
    expect(s.shape).toContain('rounded-full')
  })
  it('uses the icon name for the glyph when the action id is unknown', () => {
    const s = tileStyleFor('custom-thing', 'archive')
    expect(s.glyph).toBe(Archive)
    expect(s.shape).toContain('rounded-[14px]')
  })
  it('falls back to the File glyph when both id and icon name are unknown', () => {
    expect(tileStyleFor('custom-thing', 'mystery').glyph).toBe(File)
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover icon + tile-style resolution`

### Task 25 — `hooks/useTargetShortcuts`

- [ ] Write `apps/desktop/frontend/src/hooks/useTargetShortcuts.test.tsx`:

```tsx
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type Target } from '@/lib/backend'
import { useTargetShortcuts } from '@/hooks/useTargetShortcuts'

vi.mock('@/lib/backend')

const target = (id: string, shortcut?: string): Target => ({ id, shortcut }) as Target

function press(key: string, opts: Partial<KeyboardEventInit> = {}) {
  window.dispatchEvent(
    new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...opts }),
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('useTargetShortcuts', () => {
  it('clicks the target whose shortcut matches the key (case-insensitive)', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F'), target('t2', 'G')]))
    press('f')
    expect(backend.click).toHaveBeenCalledWith('t1')
  })
  it('ignores keystrokes held with a modifier', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    press('f', { metaKey: true })
    press('f', { ctrlKey: true })
    press('f', { altKey: true })
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('ignores keystrokes aimed at an input element', () => {
    const input = document.createElement('input')
    document.body.appendChild(input)
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'f', bubbles: true }))
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('does nothing when no shortcut matches', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    press('z')
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('ignores non-single-character keys such as Enter', () => {
    renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    press('Enter')
    expect(backend.click).not.toHaveBeenCalled()
  })
  it('detaches its keydown listener on unmount', () => {
    const { unmount } = renderHook(() => useTargetShortcuts([target('t1', 'F')]))
    unmount()
    press('f')
    expect(backend.click).not.toHaveBeenCalled()
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover keyboard-shortcut target launching`

### Task 26 — `hooks/useFileIcon`

> The hook keeps a **module-level** icon cache. Tests use a *unique path per
> case* so cache state never leaks between them.

- [ ] Write `apps/desktop/frontend/src/hooks/useFileIcon.test.tsx`:

```tsx
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend } from '@/lib/backend'
import { useFileIcon } from '@/hooks/useFileIcon'

vi.mock('@/lib/backend')

const mockIcon = vi.mocked(backend.fileIcon)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useFileIcon', () => {
  it('returns null and never calls the backend for an undefined path', () => {
    const { result } = renderHook(() => useFileIcon(undefined))
    expect(result.current).toBeNull()
    expect(mockIcon).not.toHaveBeenCalled()
  })
  it('fetches and returns the base64 icon for a path', async () => {
    mockIcon.mockResolvedValue('BASE64DATA' as never)
    const { result } = renderHook(() => useFileIcon('/unique/one.txt'))
    await waitFor(() => expect(result.current).toBe('BASE64DATA'))
    expect(mockIcon).toHaveBeenCalledWith('/unique/one.txt')
  })
  it('serves a second hook from cache without a second backend call', async () => {
    mockIcon.mockResolvedValue('CACHED' as never)
    const first = renderHook(() => useFileIcon('/unique/two.txt'))
    await waitFor(() => expect(first.result.current).toBe('CACHED'))
    mockIcon.mockClear()
    const second = renderHook(() => useFileIcon('/unique/two.txt'))
    expect(second.result.current).toBe('CACHED')
    expect(mockIcon).not.toHaveBeenCalled()
  })
  it('maps an empty backend result to null', async () => {
    mockIcon.mockResolvedValue('' as never)
    const { result } = renderHook(() => useFileIcon('/unique/three.txt'))
    await waitFor(() => expect(mockIcon).toHaveBeenCalledWith('/unique/three.txt'))
    expect(result.current).toBeNull()
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover file-icon fetch + cache`

### Task 27 — `hooks/useBackend` (targets / tasks / drop bar / specs / settings)

- [ ] Write `apps/desktop/frontend/src/hooks/useBackend.test.tsx`:

```tsx
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type Settings, type Target } from '@/lib/backend'
import { __fireEvent, __resetBackendMock, __unsub } from '@/lib/__mocks__/backend'
import {
  useActionSpecs,
  useDropBar,
  useSettings,
  useTargets,
  useTasks,
} from '@/hooks/useBackend'

vi.mock('@/lib/backend')

const target = (id: string): Target => ({ id }) as Target

beforeEach(() => {
  vi.clearAllMocks()
  __resetBackendMock()
})

describe('useTargets', () => {
  it('loads initial targets then updates on grid:changed', async () => {
    vi.mocked(backend.grid.list).mockResolvedValue([target('a')] as never)
    const { result } = renderHook(() => useTargets())
    await waitFor(() => expect(result.current).toEqual([target('a')]))
    act(() => __fireEvent('grid:changed', [target('a'), target('b')]))
    expect(result.current).toEqual([target('a'), target('b')])
  })
  it('unsubscribes from grid:changed on unmount', async () => {
    vi.mocked(backend.grid.list).mockResolvedValue([] as never)
    const { unmount } = renderHook(() => useTargets())
    await waitFor(() => expect(__unsub['grid:changed']).toBeDefined())
    unmount()
    expect(__unsub['grid:changed']).toHaveBeenCalled()
  })
})

describe('useTasks / useDropBar / useActionSpecs coerce null to []', () => {
  it('useTasks maps a null binding result to an empty array', async () => {
    vi.mocked(backend.tasks.list).mockResolvedValue(null as never)
    const { result } = renderHook(() => useTasks())
    await waitFor(() => expect(backend.tasks.list).toHaveBeenCalled())
    expect(result.current).toEqual([])
  })
  it('useDropBar updates on dropbar:changed', async () => {
    vi.mocked(backend.dropBar.list).mockResolvedValue([] as never)
    const { result } = renderHook(() => useDropBar())
    await waitFor(() => expect(backend.dropBar.list).toHaveBeenCalled())
    act(() => __fireEvent('dropbar:changed', [{ id: 'x' }]))
    expect(result.current).toEqual([{ id: 'x' }])
  })
  it('useActionSpecs updates on specs:changed', async () => {
    vi.mocked(backend.actions.specs).mockResolvedValue([] as never)
    const { result } = renderHook(() => useActionSpecs())
    await waitFor(() => expect(backend.actions.specs).toHaveBeenCalled())
    act(() => __fireEvent('specs:changed', [{ id: 'zip' }]))
    expect(result.current).toEqual([{ id: 'zip' }])
  })
})

describe('useSettings', () => {
  // Relies on the module-level settings singleton being null at first use;
  // this is the only test in the suite that consumes it.
  it('loads once, then update() persists and republishes to every consumer', async () => {
    vi.mocked(backend.settings.get).mockResolvedValue({ gridSize: 40 } as Settings as never)
    const a = renderHook(() => useSettings())
    await waitFor(() => expect(a.result.current[0]).toEqual({ gridSize: 40 }))

    // a second consumer sees the already-loaded value with no extra fetch
    vi.mocked(backend.settings.get).mockClear()
    const b = renderHook(() => useSettings())
    expect(b.result.current[0]).toEqual({ gridSize: 40 })
    expect(backend.settings.get).not.toHaveBeenCalled()

    await act(async () => {
      await a.result.current[1]({ gridSize: 80 } as Settings)
    })
    expect(backend.settings.set).toHaveBeenCalledWith({ gridSize: 80 })
    expect(a.result.current[0]).toEqual({ gridSize: 80 })
    expect(b.result.current[0]).toEqual({ gridSize: 80 })
  })
})
```

> `useSettings` writes to a module-scoped store, so it must be the sole settings
> consumer in this file (it is) and is safe under one-shot `vitest run`.

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover live backend hooks (targets/tasks/dropbar/specs/settings)`

### Task 28 — `hooks/useNativeFileDrop` (drop routing)

- [ ] Write `apps/desktop/frontend/src/hooks/useNativeFileDrop.test.tsx`:

```tsx
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend } from '@/lib/backend'
import { setUIScale } from '@/lib/dnd'
import { __emitFileDrop, __resetRuntimeStub } from '@/test/stubs/runtime'
import { useNativeFileDrop } from '@/hooks/useNativeFileDrop'

vi.mock('@/lib/backend')

beforeEach(() => {
  __resetRuntimeStub()
  setUIScale(1)
  document.body.innerHTML = ''
  vi.clearAllMocks()
  vi.restoreAllMocks()
})

function drop(dropId: string | null, paths: string[]) {
  if (dropId) {
    document.body.innerHTML = `<div data-drop-id="${dropId}"></div>`
    vi.spyOn(document, 'elementFromPoint').mockReturnValue(
      document.querySelector('[data-drop-id]'),
    )
  } else {
    vi.spyOn(document, 'elementFromPoint').mockReturnValue(null)
  }
  __emitFileDrop(10, 10, paths)
}

describe('useNativeFileDrop', () => {
  it('stashes files in the drop bar for the dropbar target', () => {
    renderHook(() => useNativeFileDrop())
    drop('dropbar', ['/a.txt'])
    expect(backend.playDropSound).toHaveBeenCalled()
    expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: 'files', paths: ['/a.txt'] })
  })
  it('adds files to the grid for the add-to-grid target', () => {
    renderHook(() => useNativeFileDrop())
    drop('add-to-grid', ['/a', '/b'])
    expect(backend.grid.addFromPaths).toHaveBeenCalledWith(['/a', '/b'])
  })
  it('runs the action and hides the window for a normal target', () => {
    renderHook(() => useNativeFileDrop())
    drop('t123', ['/a.txt'])
    expect(backend.drop).toHaveBeenCalledWith('t123', { kind: 'files', paths: ['/a.txt'] })
    expect(backend.window.hide).toHaveBeenCalled()
  })
  it('ignores drops with no target under the cursor', () => {
    renderHook(() => useNativeFileDrop())
    drop(null, ['/a'])
    expect(backend.playDropSound).not.toHaveBeenCalled()
  })
  it('ignores empty drops', () => {
    renderHook(() => useNativeFileDrop())
    drop('t123', [])
    expect(backend.playDropSound).not.toHaveBeenCalled()
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover native file-drop routing`

---

## Phase 7 — Frontend components (logic-bearing)

Every test in this phase `vi.mock('@/lib/backend')` and renders the real
component through the real shadcn `@/components/ui/*` primitives (untouched) in
jsdom. We assert **behaviour** — which backend binding fires with which
arguments, and which branch renders — not markup. Radix `Select`/`Slider`
interactions are intentionally *not* driven (jsdom has no layout / pointer
capture); we exercise them via the props/switches instead and rely on the
setup.ts polyfills only to let them *render*.

### Task 29 — `features/grid/OptionsForm`

- [ ] Write `apps/desktop/frontend/src/features/grid/OptionsForm.test.tsx`:

```tsx
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type OptionField } from '@/lib/backend'
import { OptionsForm } from '@/features/grid/OptionsForm'

vi.mock('@/lib/backend')

const field = (over: Partial<OptionField>): OptionField =>
  ({ key: 'k', label: 'K', type: 'text', ...over }) as OptionField

beforeEach(() => vi.clearAllMocks())

describe('OptionsForm', () => {
  it('reports text edits through onChange, merged into the value map', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <OptionsForm
        fields={[field({ key: 'name', label: 'Name', type: 'text' })]}
        values={{}}
        onChange={onChange}
      />,
    )
    await user.type(screen.getByRole('textbox'), 'a')
    expect(onChange).toHaveBeenCalledWith({ name: 'a' })
  })

  it('toggles a checkbox field to the string "true"', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <OptionsForm
        fields={[field({ key: 'flag', label: 'Flag', type: 'checkbox' })]}
        values={{}}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole('switch'))
    expect(onChange).toHaveBeenCalledWith({ flag: 'true' })
  })

  it('runs the folder picker and stores the chosen path', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    vi.mocked(backend.dialogs.chooseFolder).mockResolvedValue('/picked/dir' as never)
    render(
      <OptionsForm
        fields={[field({ key: 'dir', label: 'Dir', type: 'folder' })]}
        values={{}}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole('button', { name: /choose/i }))
    await waitFor(() => expect(onChange).toHaveBeenCalledWith({ dir: '/picked/dir' }))
  })

  it('uses the application picker for an app field', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    vi.mocked(backend.dialogs.chooseApplication).mockResolvedValue('/Apps/X.app' as never)
    render(
      <OptionsForm
        fields={[field({ key: 'app', label: 'App', type: 'app' })]}
        values={{}}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole('button', { name: /choose/i }))
    await waitFor(() => expect(onChange).toHaveBeenCalledWith({ app: '/Apps/X.app' }))
    expect(backend.dialogs.chooseFolder).not.toHaveBeenCalled()
  })

  it('does not store a path when the picker is cancelled', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    vi.mocked(backend.dialogs.chooseFolder).mockResolvedValue('' as never)
    render(
      <OptionsForm
        fields={[field({ key: 'dir', label: 'Dir', type: 'folder' })]}
        values={{}}
        onChange={onChange}
      />,
    )
    await user.click(screen.getByRole('button', { name: /choose/i }))
    await waitFor(() => expect(backend.dialogs.chooseFolder).toHaveBeenCalled())
    expect(onChange).not.toHaveBeenCalled()
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover OptionsForm field editing + pickers`

### Task 30 — `features/grid/AddTargetDialog`

- [ ] Write `apps/desktop/frontend/src/features/grid/AddTargetDialog.test.tsx`:

```tsx
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type ActionSpec, type Target } from '@/lib/backend'
import { AddTargetDialog } from '@/features/grid/AddTargetDialog'

vi.mock('@/lib/backend')

const spec = (over: Partial<ActionSpec>): ActionSpec =>
  ({ id: 'zip', name: 'Zip', description: 'Compress', icon: 'archive', options: [], ...over }) as ActionSpec

beforeEach(() => vi.clearAllMocks())

describe('AddTargetDialog', () => {
  it('shows the catalogue, then the config form after picking an action', async () => {
    const user = userEvent.setup()
    render(
      <AddTargetDialog
        open
        onOpenChange={vi.fn()}
        specs={[spec({ id: 'zip', name: 'Zip' }), spec({ id: 'trash', name: 'Trash' })]}
      />,
    )
    expect(screen.getByText('Add to Grid')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Zip/ }))
    expect(screen.getByText('Name in grid')).toBeInTheDocument()
  })

  it('adds a new target with the action id, label, and option map', async () => {
    const user = userEvent.setup()
    const onOpenChange = vi.fn()
    render(<AddTargetDialog open onOpenChange={onOpenChange} specs={[spec({})]} />)
    await user.click(screen.getByRole('button', { name: /Zip/ }))
    await user.click(screen.getByRole('button', { name: 'Add to Grid' }))
    await waitFor(() => expect(backend.grid.add).toHaveBeenCalledWith('zip', 'Zip', {}))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('updates an existing target in edit mode', async () => {
    const user = userEvent.setup()
    const editing = {
      id: 't1',
      actionId: 'zip',
      label: 'My Zip',
      options: {},
      shortcut: '',
    } as Target
    render(
      <AddTargetDialog open onOpenChange={vi.fn()} specs={[spec({})]} editing={editing} />,
    )
    expect(screen.getByText('Edit My Zip')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(backend.grid.update).toHaveBeenCalledWith({
        ...editing,
        label: 'My Zip',
        options: {},
        shortcut: '',
      }),
    )
  })

  it('disables submit until every required option is filled', async () => {
    const user = userEvent.setup()
    const urlSpec = spec({
      id: 'shorten-url',
      name: 'Shorten',
      options: [{ key: 'url', label: 'URL', type: 'text', required: true } as never],
    })
    render(<AddTargetDialog open onOpenChange={vi.fn()} specs={[urlSpec]} />)
    await user.click(screen.getByRole('button', { name: /Shorten/ }))
    const submit = screen.getByRole('button', { name: 'Add to Grid' })
    expect(submit).toBeDisabled()
    await user.type(screen.getAllByRole('textbox')[2], 'https://x')
    expect(submit).toBeEnabled()
  })
})
```

> The config form renders three textboxes in order — “Name in grid” `[0]`, the
> single-key “Shortcut” `[1]`, and the required “URL” option `[2]` — so the
> option field is `getAllByRole('textbox')[2]`. `getByRole('textbox', {name:''})`
> would throw on the two empty-name inputs. Typing into `[2]` flips
> `missingRequired` via `OptionsForm`'s `onChange` → `setValues`, enabling submit.

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover AddTargetDialog add/edit/validation`

### Task 31 — `features/dropbar/RenameItemDialog`

- [ ] Write `apps/desktop/frontend/src/features/dropbar/RenameItemDialog.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type DropBarItem } from '@/lib/backend'
import { RenameItemDialog } from '@/features/dropbar/RenameItemDialog'

vi.mock('@/lib/backend')

const item = { id: 'i1' } as DropBarItem

beforeEach(() => vi.clearAllMocks())

describe('RenameItemDialog', () => {
  it('is closed when value is null', () => {
    render(<RenameItemDialog item={item} value={null} onValueChange={vi.fn()} />)
    expect(screen.queryByText('Rename Item')).not.toBeInTheDocument()
  })

  it('renames on Save and closes', async () => {
    const user = userEvent.setup()
    const onValueChange = vi.fn()
    render(<RenameItemDialog item={item} value={'notes'} onValueChange={onValueChange} />)
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(backend.dropBar.rename).toHaveBeenCalledWith('i1', 'notes')
    expect(onValueChange).toHaveBeenCalledWith(null)
  })

  it('renames on Enter', async () => {
    const user = userEvent.setup()
    render(<RenameItemDialog item={item} value={'notes'} onValueChange={vi.fn()} />)
    await user.type(screen.getByRole('textbox'), '{Enter}')
    expect(backend.dropBar.rename).toHaveBeenCalledWith('i1', 'notes')
  })

  it('Reset commits an empty label so the content-derived name returns', async () => {
    const user = userEvent.setup()
    const onValueChange = vi.fn()
    render(<RenameItemDialog item={item} value={'notes'} onValueChange={onValueChange} />)
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    expect(backend.dropBar.rename).toHaveBeenCalledWith('i1', '')
    expect(onValueChange).toHaveBeenCalledWith(null)
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover RenameItemDialog save/enter/reset`

### Task 32 — `features/tasks/InputRequestDialog`

- [ ] Write `apps/desktop/frontend/src/features/tasks/InputRequestDialog.test.tsx`:

```tsx
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend } from '@/lib/backend'
import { __fireEvent, __resetBackendMock } from '@/lib/__mocks__/backend'
import { InputRequestDialog } from '@/features/tasks/InputRequestDialog'

vi.mock('@/lib/backend')

beforeEach(() => {
  vi.clearAllMocks()
  __resetBackendMock()
})

function fireRequest(req: Record<string, unknown>) {
  act(() => __fireEvent('input:request', req))
}

describe('InputRequestDialog', () => {
  it('answers a text prompt with the typed value on OK', async () => {
    const user = userEvent.setup()
    render(<InputRequestDialog />)
    fireRequest({ id: 'r1', title: 'Name', prompt: 'New name?' })
    await user.type(screen.getByRole('textbox'), 'foo')
    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(backend.answerInput).toHaveBeenCalledWith('r1', 'foo', true)
  })

  it('answers not-answered on Cancel', async () => {
    const user = userEvent.setup()
    render(<InputRequestDialog />)
    fireRequest({ id: 'r1', title: 'Name', prompt: 'New name?' })
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(backend.answerInput).toHaveBeenCalledWith('r1', '', false)
  })

  it('answers a choice prompt with the picked label', async () => {
    const user = userEvent.setup()
    render(<InputRequestDialog />)
    fireRequest({
      id: 'r2',
      title: 'File exists',
      prompt: 'a.txt already exists',
      choices: ['Keep Both', 'Replace', 'Stop'],
    })
    await user.click(screen.getByRole('button', { name: 'Replace' }))
    expect(backend.answerInput).toHaveBeenCalledWith('r2', 'Replace', true)
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover InputRequestDialog text + choice prompts`

### Task 33 — `features/tasks/TaskList`

- [ ] Write `apps/desktop/frontend/src/features/tasks/TaskList.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type ActionSpec, type Target, type TaskState } from '@/lib/backend'
import { TaskList } from '@/features/tasks/TaskList'

vi.mock('@/lib/backend')

const targets = [{ id: 't1', actionId: 'zip' } as Target]
const specFor = vi.fn(() => ({ id: 'zip', icon: 'archive' }) as ActionSpec)
const task = (over: Partial<TaskState>): TaskState =>
  ({ id: 'k1', targetId: 't1', title: 'Zipping', status: 'running', percent: 50, ...over }) as TaskState

beforeEach(() => vi.clearAllMocks())

describe('TaskList', () => {
  it('renders a running task with title and detail', () => {
    render(<TaskList tasks={[task({ detail: 'a.txt' })]} targets={targets} specFor={specFor} />)
    expect(screen.getByText('Zipping — a.txt')).toBeInTheDocument()
  })

  it('cancels a running task via the round button', async () => {
    const user = userEvent.setup()
    render(<TaskList tasks={[task({})]} targets={targets} specFor={specFor} />)
    await user.click(screen.getByTitle('Cancel'))
    expect(backend.tasks.cancel).toHaveBeenCalledWith('k1')
  })

  it('dismisses a finished task', async () => {
    const user = userEvent.setup()
    render(
      <TaskList
        tasks={[task({ status: 'done', percent: 100 })]}
        targets={targets}
        specFor={specFor}
      />,
    )
    await user.click(screen.getByTitle('Dismiss'))
    expect(backend.tasks.dismiss).toHaveBeenCalledWith('k1')
  })

  it('opens a result URL when present', async () => {
    const user = userEvent.setup()
    render(
      <TaskList
        tasks={[task({ status: 'done', percent: 100, resultUrl: 'https://x.test/a' })]}
        targets={targets}
        specFor={specFor}
      />,
    )
    await user.click(screen.getByRole('button', { name: 'https://x.test/a' }))
    expect(backend.shares.open).toHaveBeenCalledWith('https://x.test/a')
  })

  it('shows an error task as "title: error" in red', () => {
    render(
      <TaskList
        tasks={[task({ status: 'error', title: 'Zip', error: 'boom' })]}
        targets={targets}
        specFor={specFor}
      />,
    )
    const line = screen.getByText('Zip: boom')
    expect(line).toBeInTheDocument()
    expect(line.className).toContain('text-red-400')
  })

  it('caps the list at four rows', () => {
    const many = Array.from({ length: 6 }, (_, i) => task({ id: `k${i}`, title: `Task ${i}` }))
    render(<TaskList tasks={many} targets={targets} specFor={specFor} />)
    expect(screen.getAllByTitle('Cancel')).toHaveLength(4)
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover TaskList rows, cancel/dismiss, result link`

### Task 34 — `features/settings/GeneralTab`

- [ ] Write `apps/desktop/frontend/src/features/settings/GeneralTab.test.tsx`. Locate
  each toggle by its row label, then the `switch` role within that row:

```tsx
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Settings } from '@/lib/backend'
import { GeneralTab } from '@/features/settings/GeneralTab'

vi.mock('@/lib/backend')

const base = (): Settings =>
  ({
    gridSize: 33,
    gridColumns: 4,
    globalShortcut: 'F3',
    popOutShortcut: 'F4',
    theme: 'system',
    animateGrid: true,
    showKeyOverlays: false,
    launchAtLogin: false,
    notifyOnComplete: true,
    playSounds: true,
    dragOverlay: true,
    dropBarKeepsItems: false,
  }) as Settings

const switchInRow = (label: string) =>
  within(screen.getByText(label).closest('div') as HTMLElement).getByRole('switch')

beforeEach(() => vi.clearAllMocks())

describe('GeneralTab', () => {
  it('enables launch-at-login', async () => {
    const user = userEvent.setup()
    const update = vi.fn()
    render(<GeneralTab settings={base()} update={update} />)
    await user.click(switchInRow('Launch at login'))
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ launchAtLogin: true }))
  })

  it('switches to forced dark mode', async () => {
    const user = userEvent.setup()
    const update = vi.fn()
    render(<GeneralTab settings={base()} update={update} />)
    await user.click(switchInRow('Always use dark mode'))
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ theme: 'dark' }))
  })

  it('turns off play-sounds', async () => {
    const user = userEvent.setup()
    const update = vi.fn()
    render(<GeneralTab settings={base()} update={update} />)
    await user.click(switchInRow('Play sounds'))
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ playSounds: false }))
  })

  it('enables keep-drop-bar-items', async () => {
    const user = userEvent.setup()
    const update = vi.fn()
    render(<GeneralTab settings={base()} update={update} />)
    await user.click(switchInRow('Keep Drop Bar items after drag out'))
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ dropBarKeepsItems: true }))
  })
})
```

> `screen.getByText(label)` returns the row's `<Label>`; `.closest('div')` is
> the `SettingRow` wrapper that also holds the `switch`. This avoids relying on
> switch ordering.

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover GeneralTab setting toggles`

### Task 35 — `features/settings/UpdatesTab`

- [ ] Write `apps/desktop/frontend/src/features/settings/UpdatesTab.test.tsx`:

```tsx
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { backend, type Settings, type UpdateInfo } from '@/lib/backend'
import { UpdatesTab } from '@/features/settings/UpdatesTab'

vi.mock('@/lib/backend')

const settings = { autoUpdateCheck: true } as Settings
const info = (over: Partial<UpdateInfo>): UpdateInfo =>
  ({ available: false, latest: '', current: '', url: '', downloadUrl: '', ...over }) as UpdateInfo

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(backend.updates.version).mockResolvedValue('v0.3.8' as never)
})

describe('UpdatesTab', () => {
  it('checks on mount and offers the download when an update is available', async () => {
    const user = userEvent.setup()
    vi.mocked(backend.updates.check).mockResolvedValue(
      info({ available: true, latest: 'v0.4.0', url: 'https://gh/notes', downloadUrl: 'https://gh/dl' }) as never,
    )
    render(<UpdatesTab settings={settings} update={vi.fn()} />)
    await waitFor(() => expect(screen.getByText(/v0\.4\.0 is available/)).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /Download v0\.4\.0/ }))
    expect(backend.openURL).toHaveBeenCalledWith('https://gh/dl')
  })

  it('reports up-to-date when no newer version exists', async () => {
    vi.mocked(backend.updates.check).mockResolvedValue(info({ available: false }) as never)
    render(<UpdatesTab settings={settings} update={vi.fn()} />)
    await waitFor(() => expect(screen.getByText(/up to date/i)).toBeInTheDocument())
  })

  it('surfaces a check error', async () => {
    vi.mocked(backend.updates.check).mockRejectedValue(new Error('offline') as never)
    render(<UpdatesTab settings={settings} update={vi.fn()} />)
    await waitFor(() => expect(screen.getByText(/offline/)).toBeInTheDocument())
  })

  it('re-checks when Check Now is pressed', async () => {
    const user = userEvent.setup()
    vi.mocked(backend.updates.check).mockResolvedValue(info({ available: false }) as never)
    render(<UpdatesTab settings={settings} update={vi.fn()} />)
    await waitFor(() => expect(backend.updates.check).toHaveBeenCalledTimes(1))
    await user.click(screen.getByRole('button', { name: /Check Now/ }))
    await waitFor(() => expect(backend.updates.check).toHaveBeenCalledTimes(2))
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover UpdatesTab check/available/error flows`

### Task 36 — `features/onboarding/Onboarding`

- [ ] Write `apps/desktop/frontend/src/features/onboarding/Onboarding.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Onboarding } from '@/features/onboarding/Onboarding'

describe('Onboarding', () => {
  it('starts on the welcome slide with Back hidden', () => {
    render(<Onboarding onDone={vi.fn()} />)
    expect(screen.getByText('Welcome to DragZone')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Back/ })).toHaveClass('invisible')
  })

  it('advances through slides with Next', async () => {
    const user = userEvent.setup()
    render(<Onboarding onDone={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /Next/ }))
    expect(screen.getByText('Drop files onto actions')).toBeInTheDocument()
  })

  it('jumps to a slide via its dot', async () => {
    const user = userEvent.setup()
    render(<Onboarding onDone={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: 'Slide 3' }))
    expect(screen.getByText('Stash in the Drop Bar')).toBeInTheDocument()
  })

  it('calls onDone from Skip', async () => {
    const user = userEvent.setup()
    const onDone = vi.fn()
    render(<Onboarding onDone={onDone} />)
    await user.click(screen.getByRole('button', { name: 'Skip' }))
    expect(onDone).toHaveBeenCalled()
  })

  it('calls onDone from Get Started on the last slide', async () => {
    const user = userEvent.setup()
    const onDone = vi.fn()
    render(<Onboarding onDone={onDone} />)
    await user.click(screen.getByRole('button', { name: 'Slide 5' }))
    await user.click(screen.getByRole('button', { name: 'Get Started' }))
    expect(onDone).toHaveBeenCalled()
  })
})
```

- [ ] Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test` — Expected: PASS.
- [ ] Commit: `test(frontend): cover Onboarding carousel navigation`

---

## Phase 8 — Full-suite green gate

A single task that proves the whole tree is green from a clean checkout, in the
exact order a fresh contributor (or CI) would run it, and records the outcome
in the design spec so the coverage claim is verifiable.

### Task 37 — Run everything, clean-tree, and record the result

- [ ] Run (desktop backend — requires the embed to exist first):

```sh
cd apps/desktop
wails build            # regenerates frontend/dist + wailsjs so `go test` on package main compiles
go test ./...          # every backend package + the App facade
gofmt -l .             # MUST print nothing
```

- [ ] Run (desktop frontend):

```sh
cd apps/desktop
bun run --filter=@dragzone/desktop-frontend test   # full vitest suite
cd frontend && bunx tsc --noEmit                    # build tsconfig stays green (tests excluded)
```

- [ ] Run (repo-root workspaces — web + shared, must stay untouched/green):

```sh
cd "$(git rev-parse --show-toplevel)"
bun run test           # vitest for apps/web + packages/shared
bun run lint           # biome check web + shared
```

- [ ] Confirm: every command above exits 0; `gofmt -l .` and `bunx tsc --noEmit`
  print nothing. If any Go test wrote outside `t.TempDir()`, re-audit for a
  missing `t.Setenv(storage.EnvDataDir, t.TempDir())`.
- [ ] Update `docs/superpowers/specs/2026-07-07-test-coverage-design.md`:
  check off the verification checklist and note the final package/file count
  covered.
- [ ] Commit: `test: green-gate the full backend + frontend + workspace suites`

---

## Done-when

- Every Go package under `apps/desktop` (model, storage, config, ipc, actions
  registry + all builtin actions, tasks, bundles) and the `App` facade
  (grid/dropbar/ipc/shares/settings) has direct unit tests; `go test ./...`
  passes and `gofmt -l .` is empty.
- Network actions (imgur, shorten, addons, s3, gdrive) hit `httptest` servers
  through injectable-endpoint `var`s — no live third-party I/O.
- The desktop frontend has a working vitest + jsdom + Testing Library harness
  with a manual `@/lib/backend` mock; `lib/`, `hooks/`, and every logic-bearing
  feature component is covered; the frontend suite passes and `tsc --noEmit`
  stays green.
- `apps/web` and `packages/shared` suites/lint remain green and untouched.
- Native cgo (`internal/platform`), shadcn `components/ui/*`, and purely
  presentational wrappers remain intentionally out of scope (documented in the
  design spec).
