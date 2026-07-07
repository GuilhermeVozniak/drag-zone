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
	if _, err := (SaveText{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemText, Text: "x"},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("missing folder should error")
	}
	if _, err := (SaveText{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"path": t.TempDir()}},
		Payload:  model.Payload{Kind: model.ItemText, Text: "   "},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("blank text should error")
	}
}
