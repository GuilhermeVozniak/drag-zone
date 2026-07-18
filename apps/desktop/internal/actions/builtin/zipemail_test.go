package builtin

import (
	"archive/zip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// withFakeMailCmd swaps mailCmd for a fake that records the script it was
// invoked with, running a no-op command so no real Mail.app window opens.
// Restores the original mailCmd on cleanup.
func withFakeMailCmd(t *testing.T) *[]string {
	t.Helper()
	var scripts []string
	orig := mailCmd
	t.Cleanup(func() { mailCmd = orig })
	mailCmd = func(ctx context.Context, script string) *exec.Cmd {
		scripts = append(scripts, script)
		return exec.CommandContext(ctx, "true")
	}
	return &scripts
}

func TestZipEmailSpec(t *testing.T) {
	spec := ZipEmail{}.Spec()
	if spec.ID != "zip-email" {
		t.Errorf("ID = %q, want %q", spec.ID, "zip-email")
	}
	if spec.Name != "Zip & Email" {
		t.Errorf("Name = %q, want %q", spec.Name, "Zip & Email")
	}
	if spec.Icon != "mail" {
		t.Errorf("Icon = %q, want %q", spec.Icon, "mail")
	}
	if spec.Category != "File Management" {
		t.Errorf("Category = %q, want %q", spec.Category, "File Management")
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventDragged {
		t.Errorf("Events = %v, want [%s]", spec.Events, model.EventDragged)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %v, want [%s]", spec.Accepts, model.ItemFiles)
	}
	if spec.Multi {
		t.Error("expected Multi = false")
	}
}

func TestZipEmailEmptyPayloadErrorsBeforeZipping(t *testing.T) {
	scripts := withFakeMailCmd(t)
	_, err := ZipEmail{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	})
	if err == nil {
		t.Fatal("expected an error for an empty payload")
	}
	if len(*scripts) != 0 {
		t.Errorf("osascript should not be invoked on an empty payload, got %v", *scripts)
	}
}

func TestZipEmailZipsAndComposesEmail(t *testing.T) {
	scripts := withFakeMailCmd(t)

	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(fileA, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ZipEmail{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{fileA, fileB}},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "Zipped 2 file(s) into an email" {
		t.Errorf("message = %q, want %q", res.Message, "Zipped 2 file(s) into an email")
	}

	if len(*scripts) != 1 {
		t.Fatalf("expected 1 osascript call, got %d: %v", len(*scripts), *scripts)
	}
	script := (*scripts)[0]

	if !strings.Contains(script, `tell application "Mail"`) {
		t.Errorf("script = %q, want it to target Mail", script)
	}
	if !strings.Contains(script, "make new outgoing message") {
		t.Errorf("script = %q, want it to create an outgoing message", script)
	}
	if !strings.Contains(script, "make new attachment") {
		t.Errorf("script = %q, want it to attach the zip", script)
	}

	// Extract the zip path referenced in the script (inside the
	// POSIX file "<path>" clause) and verify it's a real, valid archive
	// containing both dropped files.
	const marker = `POSIX file `
	idx := strings.Index(script, marker)
	if idx == -1 {
		t.Fatalf("script = %q, want a POSIX file reference", script)
	}
	rest := script[idx+len(marker):]
	if !strings.HasPrefix(rest, `"`) {
		t.Fatalf("script = %q, want a quoted path after POSIX file", script)
	}
	end := strings.Index(rest[1:], `"`)
	if end == -1 {
		t.Fatalf("script = %q, want a closing quote for the path", script)
	}
	zipPath := rest[1 : 1+end]

	if !strings.HasSuffix(zipPath, ".zip") {
		t.Errorf("zipPath = %q, want a .zip file", zipPath)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("expected zip file to exist at %q: %v", zipPath, err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("zip file is not a valid archive: %v", err)
	}
	defer r.Close()
	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["a.txt"] || !names["b.txt"] {
		t.Errorf("zip contents = %v, want a.txt and b.txt", names)
	}
}

func TestZipEmailMailCmdErrorWrapped(t *testing.T) {
	orig := mailCmd
	t.Cleanup(func() { mailCmd = orig })
	mailCmd = func(ctx context.Context, script string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(src, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ZipEmail{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
	})
	if err == nil {
		t.Fatal("expected an error when osascript fails")
	}
	if !strings.Contains(err.Error(), "composing email") {
		t.Errorf("error = %q, want it to mention %q", err.Error(), "composing email")
	}
}
