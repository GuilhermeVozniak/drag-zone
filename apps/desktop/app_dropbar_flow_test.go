package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"dragzone/internal/dropbar"
	"dragzone/internal/model"
)

// dropBarEmitRecorder captures every EventDropBarChanged emission so tests
// can assert stash/consume mutations actually notify the frontend, not just
// that the store mutated. app.onEmit must be wired to this recorder BEFORE
// any mutating call: app.emit (app.go) falls back to the real Wails
// runtime.EventsEmit whenever onEmit is nil, which aborts the test process
// since these tests never call startup() to seed a real runtime context.
type dropBarEmitRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *dropBarEmitRecorder) record(ev string, _ ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *dropBarEmitRecorder) count(ev string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e == ev {
			n++
		}
	}
	return n
}

// TestDropBarStashConsumeFlow exercises the Drop Bar stash -> consume path
// that a real drag-out/drop completion drives (DropBarConsume is called by
// StartDragOut and by DropOnTarget after a successful drop; native drag
// itself is not unit-testable, see app_dropbar.go StartDragOut). It asserts,
// in the order Dropzone's own lifecycle produces them:
//  1. stashing two file payloads emits EventDropBarChanged and both land in
//     DropBarItems();
//  2. consuming an unlocked item (keep-items off) removes it;
//  3. consuming a locked item retains it;
//  4. consuming an unlocked item while DropBarKeepsItems is on retains it.
func TestDropBarStashConsumeFlow(t *testing.T) {
	app := newTestApp(t)

	rec := &dropBarEmitRecorder{}
	app.onEmit = rec.record

	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	for _, p := range []string{fileA, fileB} {
		if err := os.WriteFile(p, []byte("payload"), 0o644); err != nil {
			t.Fatalf("seed %s: %v", p, err)
		}
	}

	// 1. Stashing two payloads emits EventDropBarChanged and both appear.
	itemA, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{fileA}})
	if err != nil {
		t.Fatalf("DropBarAdd(a): %v", err)
	}
	itemB, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{fileB}})
	if err != nil {
		t.Fatalf("DropBarAdd(b): %v", err)
	}

	if got := rec.count(EventDropBarChanged); got != 2 {
		t.Errorf("EventDropBarChanged emitted %d times after two adds, want 2", got)
	}
	if got := len(app.DropBarItems()); got != 2 {
		t.Fatalf("DropBarItems() len = %d, want 2", got)
	}

	// 2. Unlocked item, keep-items off: consume removes it.
	if err := app.DropBarConsume(itemA.ID); err != nil {
		t.Fatalf("DropBarConsume(itemA): %v", err)
	}
	if containsItem(app.DropBarItems(), itemA.ID) {
		t.Error("unlocked item survived DropBarConsume with keep-items off")
	}
	if got := len(app.DropBarItems()); got != 1 {
		t.Errorf("DropBarItems() len after consuming itemA = %d, want 1", got)
	}

	// 3. Locked item: consume retains it.
	if err := app.DropBarSetLocked(itemB.ID, true); err != nil {
		t.Fatalf("DropBarSetLocked(itemB, true): %v", err)
	}
	if err := app.DropBarConsume(itemB.ID); err != nil {
		t.Fatalf("DropBarConsume(itemB locked): %v", err)
	}
	if !containsItem(app.DropBarItems(), itemB.ID) {
		t.Error("locked item was removed by DropBarConsume")
	}

	// 4. keep-items on: consume retains an unlocked item.
	s := app.settings.Get()
	s.DropBarKeepsItems = true
	if err := app.settings.Set(s); err != nil {
		t.Fatalf("settings.Set(DropBarKeepsItems=true): %v", err)
	}

	fileC := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(fileC, []byte("payload"), 0o644); err != nil {
		t.Fatalf("seed %s: %v", fileC, err)
	}
	itemC, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{fileC}})
	if err != nil {
		t.Fatalf("DropBarAdd(c): %v", err)
	}
	if err := app.DropBarConsume(itemC.ID); err != nil {
		t.Fatalf("DropBarConsume(itemC, keep-items on): %v", err)
	}
	if !containsItem(app.DropBarItems(), itemC.ID) {
		t.Error("unlocked item was removed by DropBarConsume while DropBarKeepsItems is true")
	}
}

func containsItem(items []dropbar.Item, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}
