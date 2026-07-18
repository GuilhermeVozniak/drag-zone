package builtin

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// withFakeNoteCmd swaps noteCmd for a fake that records the script it was
// invoked with, running a no-op command so no real Apple Note is created.
// Restores the original noteCmd on cleanup.
func withFakeNoteCmd(t *testing.T) *[]string {
	t.Helper()
	var scripts []string
	orig := noteCmd
	t.Cleanup(func() { noteCmd = orig })
	noteCmd = func(ctx context.Context, script string) *exec.Cmd {
		scripts = append(scripts, script)
		return exec.CommandContext(ctx, "true")
	}
	return &scripts
}

func TestAppleNoteSpec(t *testing.T) {
	spec := AppleNote{}.Spec()
	if spec.ID != "apple-note" {
		t.Errorf("ID = %q, want %q", spec.ID, "apple-note")
	}
	if spec.Name != "Create Apple Note" {
		t.Errorf("Name = %q, want %q", spec.Name, "Create Apple Note")
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventDragged {
		t.Errorf("Events = %v, want [%s]", spec.Events, model.EventDragged)
	}
	want := []model.ItemKind{model.ItemText, model.ItemFiles}
	if len(spec.Accepts) != len(want) || spec.Accepts[0] != want[0] || spec.Accepts[1] != want[1] {
		t.Errorf("Accepts = %v, want %v", spec.Accepts, want)
	}
	if spec.Multi {
		t.Error("expected Multi = false")
	}
	if spec.Icon != "notebook-pen" {
		t.Errorf("Icon = %q, want %q", spec.Icon, "notebook-pen")
	}
}

func TestAppleNoteFromText(t *testing.T) {
	scripts := withFakeNoteCmd(t)
	res, err := AppleNote{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemText, Text: "Meeting notes"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "Note created" {
		t.Errorf("message = %q, want %q", res.Message, "Note created")
	}
	if len(*scripts) != 1 {
		t.Fatalf("expected 1 osascript call, got %d: %v", len(*scripts), *scripts)
	}
	if !strings.Contains((*scripts)[0], "Meeting notes") {
		t.Errorf("script = %q, want it to contain the dropped text", (*scripts)[0])
	}
	if !strings.Contains((*scripts)[0], `tell application "Notes" to make new note`) {
		t.Errorf("script = %q, want the Notes make-new-note command", (*scripts)[0])
	}
}

func TestAppleNoteFromFiles(t *testing.T) {
	scripts := withFakeNoteCmd(t)
	res, err := AppleNote{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemFiles, Paths: []string{"/a/one.txt", "/b/two.txt"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "Note created" {
		t.Errorf("message = %q, want %q", res.Message, "Note created")
	}
	if len(*scripts) != 1 {
		t.Fatalf("expected 1 osascript call, got %d: %v", len(*scripts), *scripts)
	}
	script := (*scripts)[0]
	if !strings.Contains(script, "one.txt") || !strings.Contains(script, "two.txt") {
		t.Errorf("script = %q, want it to mention both file base names", script)
	}
	if strings.Contains(script, "/a/one.txt") {
		t.Errorf("script = %q, want base names only, not full paths", script)
	}
}

func TestAppleNoteEmptyPayloadErrors(t *testing.T) {
	scripts := withFakeNoteCmd(t)
	_, err := AppleNote{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemText, Text: "   "},
	})
	if err == nil {
		t.Error("expected an error for blank text")
	}
	if len(*scripts) != 0 {
		t.Errorf("osascript should not be invoked on an empty payload, got %v", *scripts)
	}
}

func TestAppleNoteEmptyFilesPayloadErrors(t *testing.T) {
	scripts := withFakeNoteCmd(t)
	_, err := AppleNote{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemFiles},
	})
	if err == nil {
		t.Error("expected an error for no dropped files")
	}
	if len(*scripts) != 0 {
		t.Errorf("osascript should not be invoked on an empty payload, got %v", *scripts)
	}
}

func TestAppleNoteEscapesQuotesAndBackslashes(t *testing.T) {
	scripts := withFakeNoteCmd(t)
	_, err := AppleNote{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemText, Text: `She said "hi" \ bye`},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	script := (*scripts)[0]
	if !strings.Contains(script, `\"hi\"`) {
		t.Errorf("script = %q, want escaped double quotes", script)
	}
	if !strings.Contains(script, `\\`) {
		t.Errorf("script = %q, want an escaped backslash", script)
	}
}

func TestAppleNoteCmdErrorWrapped(t *testing.T) {
	orig := noteCmd
	t.Cleanup(func() { noteCmd = orig })
	noteCmd = func(ctx context.Context, script string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}
	_, err := AppleNote{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemText, Text: "hello"},
	})
	if err == nil {
		t.Fatal("expected an error when osascript fails")
	}
	if !strings.Contains(err.Error(), "creating note") {
		t.Errorf("error = %q, want it to mention %q", err.Error(), "creating note")
	}
}

func TestAppleScriptStringHelper(t *testing.T) {
	got := appleScriptString(`a "quote" and \ backslash`)
	want := `"a \"quote\" and \\ backslash"`
	if got != want {
		t.Errorf("appleScriptString(...) = %q, want %q", got, want)
	}
}
