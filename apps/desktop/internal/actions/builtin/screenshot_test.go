package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// withFakeScreenshotCmd swaps screenshotCmd for a fake that records the args
// it was invoked with and, when writeFile is true, actually writes a dummy
// file at the destination path (the last argument in every mode) so
// os.Stat(dst) succeeds afterward — simulating a completed capture. When
// writeFile is false, it runs a no-op command, simulating a capture the user
// cancelled (Esc), where screencapture never creates a file. Restores the
// original screenshotCmd on cleanup.
func withFakeScreenshotCmd(t *testing.T, writeFile bool) *[][]string {
	t.Helper()
	var calls [][]string
	orig := screenshotCmd
	t.Cleanup(func() { screenshotCmd = orig })
	screenshotCmd = func(ctx context.Context, args ...string) *exec.Cmd {
		calls = append(calls, append([]string(nil), args...))
		if !writeFile {
			return exec.CommandContext(ctx, "true")
		}
		dst := args[len(args)-1]
		return exec.CommandContext(ctx, "sh", "-c", "printf fakepng > \"$1\"", "--", dst)
	}
	return &calls
}

// recDropBar records paths passed to a stubbed Invocation.AddDropBar.
type recDropBar struct {
	calls [][]string
}

func (r *recDropBar) add(paths []string) { r.calls = append(r.calls, paths) }

func TestScreenshotSpec(t *testing.T) {
	spec := (Screenshot{}).Spec()
	if spec.ID != "screenshot" {
		t.Errorf("ID = %q, want %q", spec.ID, "screenshot")
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventClicked {
		t.Errorf("Events = %v, want [%s]", spec.Events, model.EventClicked)
	}
	if spec.KeyModifier != "option" {
		t.Errorf("KeyModifier = %q, want %q", spec.KeyModifier, "option")
	}
	if !spec.Multi {
		t.Error("expected Multi = true")
	}
	if spec.Icon != "camera" {
		t.Errorf("Icon = %q, want %q", spec.Icon, "camera")
	}

	byKey := map[string]model.OptionField{}
	for _, o := range spec.Options {
		byKey[o.Key] = o
	}
	mode, ok := byKey["mode"]
	if !ok || mode.Default != "interactive" {
		t.Errorf("mode option = %+v, want default interactive", mode)
	}
	wantModeChoices := []string{"interactive", "window", "screen"}
	if len(mode.Choices) != len(wantModeChoices) {
		t.Errorf("mode choices = %v, want %v", mode.Choices, wantModeChoices)
	}
	folder, ok := byKey["folder"]
	if !ok || folder.Type != "folder" {
		t.Errorf("folder option = %+v, want Type=folder", folder)
	}
	after, ok := byKey["after"]
	if !ok || after.Default != "dropbar" {
		t.Errorf("after option = %+v, want default dropbar", after)
	}
}

func TestScreenshotClickedBuildsArgsPerMode(t *testing.T) {
	cases := []struct {
		mode     string
		wantHead []string
	}{
		{"interactive", []string{"-i"}},
		{"window", []string{"-w"}},
		{"screen", nil},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			dir := t.TempDir()
			calls := withFakeScreenshotCmd(t, true)
			drop := &recDropBar{}

			fixed := time.Date(2026, 7, 18, 9, 30, 0, 0, time.UTC)
			origNow := screenshotNow
			screenshotNow = func() time.Time { return fixed }
			t.Cleanup(func() { screenshotNow = origNow })

			inv := actions.Invocation{
				Target: model.Target{Options: map[string]string{
					"mode":   tc.mode,
					"folder": dir,
				}},
				Services:   &recServices{},
				AddDropBar: drop.add,
			}

			res, err := (Screenshot{}).Clicked(context.Background(), inv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(*calls) != 1 {
				t.Fatalf("expected 1 screenshotCmd call, got %d: %+v", len(*calls), *calls)
			}
			args := (*calls)[0]
			wantName := "Screenshot 2026-07-18 at 09.30.00.png"
			wantDst := filepath.Join(dir, wantName)

			wantArgs := append(append([]string(nil), tc.wantHead...), wantDst)
			if len(args) != len(wantArgs) {
				t.Fatalf("args = %v, want %v", args, wantArgs)
			}
			for i, a := range args {
				if a != wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q (full: %v)", i, a, wantArgs[i], args)
				}
			}

			if _, err := os.Stat(dir); err != nil {
				t.Errorf("save dir was not created: %v", err)
			}
			if len(drop.calls) != 1 || len(drop.calls[0]) != 1 || drop.calls[0][0] != wantDst {
				t.Errorf("AddDropBar calls = %v, want [[%s]]", drop.calls, wantDst)
			}
			if res.Message == "" {
				t.Error("expected a non-empty result message")
			}
		})
	}
}

func TestScreenshotClickedCreatesMissingFolder(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "shots")
	withFakeScreenshotCmd(t, true)

	inv := actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"mode":   "screen",
			"folder": dir,
		}},
		Services: &recServices{},
	}

	if _, err := (Screenshot{}).Clicked(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("expected save dir %s to be created, err=%v", dir, err)
	}
}

func TestScreenshotClickedAfterReveal(t *testing.T) {
	dir := t.TempDir()
	withFakeScreenshotCmd(t, true)
	drop := &recDropBar{}
	svc := &recServices{}

	inv := actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"mode":   "screen",
			"folder": dir,
			"after":  "reveal",
		}},
		Services:   svc,
		AddDropBar: drop.add,
	}

	if _, err := (Screenshot{}).Clicked(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drop.calls) != 0 {
		t.Errorf("AddDropBar must not be called when after=reveal, got %v", drop.calls)
	}
	if len(svc.Opened) != 1 {
		t.Errorf("expected Reveal to be called once via Services, got %v", svc.Opened)
	}
}

func TestScreenshotClickedCancelledCaptureSkipsDropBarAndError(t *testing.T) {
	dir := t.TempDir()
	withFakeScreenshotCmd(t, false) // simulate Esc: screencapture writes nothing
	drop := &recDropBar{}
	svc := &recServices{}

	inv := actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"mode":   "interactive",
			"folder": dir,
		}},
		Services:   svc,
		AddDropBar: drop.add,
	}

	res, err := (Screenshot{}).Clicked(context.Background(), inv)
	if err != nil {
		t.Fatalf("cancelled capture must not be an error, got %v", err)
	}
	if res.Message != "Screenshot cancelled" {
		t.Errorf("Message = %q, want %q", res.Message, "Screenshot cancelled")
	}
	if len(drop.calls) != 0 {
		t.Errorf("AddDropBar must not be called on a cancelled capture, got %v", drop.calls)
	}
	if len(svc.Opened) != 0 {
		t.Errorf("Reveal must not be called on a cancelled capture, got %v", svc.Opened)
	}
}
