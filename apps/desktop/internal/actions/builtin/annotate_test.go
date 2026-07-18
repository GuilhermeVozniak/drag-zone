package builtin

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// stubAnnotateCmd swaps annotateCmd for one that records the args it is called
// with and runs `ok` (true) or a failing command (false, mimicking `open`
// exiting non-zero when CleanShot X isn't installed). Restored via t.Cleanup.
func stubAnnotateCmd(t *testing.T, ok bool) *[][]string {
	t.Helper()
	orig := annotateCmd
	t.Cleanup(func() { annotateCmd = orig })
	var calls [][]string
	bin := "true"
	if !ok {
		bin = "false"
	}
	annotateCmd = func(ctx context.Context, args ...string) *exec.Cmd {
		calls = append(calls, args)
		return exec.CommandContext(ctx, bin)
	}
	return &calls
}

func dropImages(t *testing.T, paths ...string) (actions.Result, error) {
	t.Helper()
	return Annotate{}.Dropped(context.Background(), actions.Invocation{
		Payload: model.Payload{Kind: model.ItemFiles, Paths: paths},
	})
}

func TestAnnotateOpensImageInCleanShot(t *testing.T) {
	calls := stubAnnotateCmd(t, true)
	res, err := dropImages(t, "/tmp/a.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 open call, got %d", len(*calls))
	}
	if got := (*calls)[0][0]; !strings.HasPrefix(got, "cleanshot://open-annotate?filepath=") {
		t.Errorf("unexpected URL: %q", got)
	}
	if !strings.Contains(res.Message, "CleanShot X") {
		t.Errorf("expected success message, got %q", res.Message)
	}
}

func TestAnnotateSkipsNonImages(t *testing.T) {
	calls := stubAnnotateCmd(t, true)
	res, err := dropImages(t, "/tmp/notes.txt", "/tmp/archive.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("expected no open calls for non-images, got %d", len(*calls))
	}
	if !strings.Contains(res.Message, "No image files") {
		t.Errorf("expected no-images message, got %q", res.Message)
	}
}

func TestAnnotateFiltersToImagesOnly(t *testing.T) {
	calls := stubAnnotateCmd(t, true)
	if _, err := dropImages(t, "/tmp/a.png", "/tmp/b.txt", "/tmp/c.JPG"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("expected 2 open calls (png + JPG), got %d", len(*calls))
	}
}

func TestAnnotateReportsCleanShotMissing(t *testing.T) {
	stubAnnotateCmd(t, false) // `open` fails => CleanShot X not installed
	res, err := dropImages(t, "/tmp/a.png")
	if err != nil {
		t.Fatalf("expected no error (friendly message), got %v", err)
	}
	if res.Message != cleanShotRequiredMessage {
		t.Errorf("expected CleanShot-required message, got %q", res.Message)
	}
}

func TestCleanShotAnnotateURLEncoding(t *testing.T) {
	got := cleanShotAnnotateURL("/Users/j/Desktop/my screenshot.png")
	want := "cleanshot://open-annotate?filepath=/Users/j/Desktop/my%20screenshot.png"
	if got != want {
		t.Errorf("URL encoding\n got: %q\nwant: %q", got, want)
	}
	// Path separators stay raw; spaces become %20 (CleanShot's documented form).
	if strings.Contains(got, "%2F") || strings.Contains(got, "+") {
		t.Errorf("slashes/spaces mis-encoded: %q", got)
	}
}
