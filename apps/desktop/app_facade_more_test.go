package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dragzone/internal/model"
)

// TestDropBarAddUnstagesOnSaveFailure: when persisting the new item fails,
// the copies StagePaths just made must be deleted, not leaked in the stage
// dir (and the item must not linger in memory — dropbar.Store.Add rolls
// back).
func TestDropBarAddUnstagesOnSaveFailure(t *testing.T) {
	app := newTestApp(t)
	// Break dropbar persistence: a directory where dropbar.json should be
	// (storage.Save's atomic rename onto a directory fails).
	dir, err := dropbarStageDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "dropbar.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	files := tempFiles(t, "leak.txt")
	if _, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: files}); err == nil {
		t.Fatal("DropBarAdd should report the save failure")
	}
	if got := app.DropBarItems(); len(got) != 0 {
		t.Errorf("failed add left items: %+v", got)
	}
	staged, err := os.ReadDir(filepath.Join(dir, "Staged"))
	if err != nil {
		if os.IsNotExist(err) {
			return // nothing staged at all — also fine
		}
		t.Fatal(err)
	}
	if len(staged) != 0 {
		t.Errorf("staged copies leaked after failed add: %+v", staged)
	}
}

func dropbarStageDataDir() (string, error) {
	// The data dir is DRAGZONE_DATA_DIR (set by newTestApp).
	return os.Getenv("DRAGZONE_DATA_DIR"), nil
}

func TestConsoleBufferCapAndClear(t *testing.T) {
	app := newTestApp(t)

	var mu sync.Mutex
	emits := 0
	app.onEmit = func(ev string, _ ...any) {
		if ev == EventConsoleChanged {
			mu.Lock()
			emits++
			mu.Unlock()
		}
	}

	for range consoleCap + 50 {
		app.appendConsole("line")
	}
	lines := app.ConsoleLines()
	if len(lines) != consoleCap {
		t.Errorf("console buffer = %d lines, want cap %d", len(lines), consoleCap)
	}
	if lines[0].Line != "line" || lines[len(lines)-1].At.IsZero() {
		t.Errorf("console lines malformed: %+v", lines[0])
	}
	// Mutating the returned slice must not corrupt the buffer.
	lines[0].Line = "corrupted"
	if app.ConsoleLines()[0].Line != "line" {
		t.Error("ConsoleLines aliases the internal buffer")
	}

	app.ClearConsole()
	if got := app.ConsoleLines(); len(got) != 0 {
		t.Errorf("after ClearConsole: %+v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if emits == 0 {
		t.Error("no console:changed emitted")
	}
}

// nextInputRequest records the IDs of emitted input requests.
func recordInputRequests(app *App) func() []string {
	var mu sync.Mutex
	var ids []string
	app.onEmit = func(ev string, data ...any) {
		if ev != EventInputRequest {
			return
		}
		if req, ok := data[0].(inputRequest); ok {
			mu.Lock()
			ids = append(ids, req.ID)
			mu.Unlock()
		}
	}
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), ids...)
	}
}

func TestPromptUserAnswer(t *testing.T) {
	app := newTestApp(t)
	ids := recordInputRequests(app)

	type answer struct {
		value string
		ok    bool
	}
	resCh := make(chan answer, 1)
	go func() {
		v, ok := app.promptUser(context.Background(), "Title", "Prompt", nil)
		resCh <- answer{v, ok}
	}()

	deadline := time.After(2 * time.Second)
	var id string
	for id == "" {
		if got := ids(); len(got) > 0 {
			id = got[0]
			break
		}
		select {
		case <-deadline:
			t.Fatal("no input:request emitted")
		case <-time.After(5 * time.Millisecond):
		}
	}
	app.AnswerInputRequest(id, "hello", true)

	select {
	case res := <-resCh:
		if !res.ok || res.value != "hello" {
			t.Errorf("promptUser = %q, %v; want hello, true", res.value, res.ok)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("promptUser did not return after AnswerInputRequest")
	}

	// A late answer for the consumed request must not panic or block.
	app.AnswerInputRequest(id, "late", true)
}

func TestPromptUserContextCancel(t *testing.T) {
	app := newTestApp(t)
	recordInputRequests(app)

	ctx, cancel := context.WithCancel(context.Background())
	resCh := make(chan bool, 1)
	go func() {
		_, ok := app.promptUser(ctx, "T", "P", []string{"Keep", "Skip"})
		resCh <- ok
	}()
	time.Sleep(20 * time.Millisecond) // let the request register
	cancel()
	select {
	case ok := <-resCh:
		if ok {
			t.Error("cancelled prompt should return ok=false")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("promptUser did not unblock on context cancel")
	}
	if got := len(app.inputReqs); got != 0 {
		t.Errorf("input request leaked after cancel: %d pending", got)
	}
}

func TestParseFKey(t *testing.T) {
	cases := map[string]int{
		"F3": 3, "f4": 4, " F12 ": 12, "F1": 1,
		"": 0, "F0": 0, "F13": 0, "F": 0, "A3": 0, "Fx": 0, "Cmd-F3": 0,
	}
	for in, want := range cases {
		if got := parseFKey(in); got != want {
			t.Errorf("parseFKey(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestGridFacadeBasics(t *testing.T) {
	app := newTestApp(t)

	if specs := app.ActionSpecs(); len(specs) == 0 {
		t.Fatal("ActionSpecs empty; built-ins not registered")
	}

	if _, err := app.AddTarget("no-such-action", "X", nil); err == nil {
		t.Error("AddTarget with unknown action should error")
	}

	a, err := app.AddTarget("folder", "A", map[string]string{"path": "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := app.AddTarget("zip", "B", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Move B ahead of A (post-removal index semantics, like the drag UI).
	// The grid is seeded with default targets, so compare relative order.
	if err := app.MoveTarget(b.ID, 0); err != nil {
		t.Fatal(err)
	}
	posOf := map[string]int{}
	for _, tg := range app.Targets() {
		posOf[tg.ID] = tg.Position
	}
	if posOf[b.ID] >= posOf[a.ID] {
		t.Errorf("after move: B at %d, A at %d; want B before A", posOf[b.ID], posOf[a.ID])
	}
	if posOf[b.ID] != 0 {
		t.Errorf("B should be first after MoveTarget(_, 0), at %d", posOf[b.ID])
	}
	if err := app.MoveTarget("missing", 0); err == nil {
		t.Error("MoveTarget with unknown id should error")
	}

	if err := app.RemoveTarget(a.ID); err != nil {
		t.Fatal(err)
	}
	for _, tg := range app.Targets() {
		if tg.ID == a.ID {
			t.Error("removed target still listed")
		}
	}
	if err := app.RemoveTarget("missing"); err != nil {
		t.Errorf("removing a missing target should be a no-op, got %v", err)
	}
}

// TestClickTargetAndTaskLifecycle clicks a Folder target (its Clicked opens
// the folder via Services, a no-op here), then dismisses the finished task.
func TestClickTargetAndTaskLifecycle(t *testing.T) {
	app := newTestApp(t)
	app.ctx = context.Background()
	// Keep emits off the real Wails runtime (a plain Background context is
	// not a valid EventsEmit target).
	app.onEmit = func(string, ...any) {}

	tg, err := app.AddTarget("folder", "F", map[string]string{"path": t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := app.ClickTarget(tg.ID)
	if err != nil {
		t.Fatalf("ClickTarget: %v", err)
	}
	if taskID == "" {
		t.Fatal("ClickTarget returned no task id")
	}

	deadline := time.After(3 * time.Second)
	for {
		list := app.Tasks()
		if len(list) == 1 && list[0].Status != model.TaskRunning {
			if list[0].Status != model.TaskDone {
				t.Fatalf("click task failed: %+v", list[0])
			}
			app.DismissTask(list[0].ID)
			if got := app.Tasks(); len(got) != 0 {
				t.Errorf("after dismiss: %+v", got)
			}
			return
		}
		select {
		case <-deadline:
			t.Fatalf("task never finished: %+v", app.Tasks())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestClickTargetRejectsDropOnlyAction(t *testing.T) {
	app := newTestApp(t)
	app.ctx = context.Background()
	app.onEmit = func(string, ...any) {}

	tg, err := app.AddTarget("zip", "Z", nil) // zip supports dragged only
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ClickTarget(tg.ID); err == nil {
		t.Error("clicking a drop-only action should error")
	}
}
