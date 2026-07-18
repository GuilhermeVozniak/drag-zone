package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// screenshotCmd is a seam so tests stub the capture (screencapture shows UI).
var screenshotCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "screencapture", args...)
}

// screenshotNow is a seam for deterministic filenames in tests.
var screenshotNow = time.Now

// Screenshot captures the screen via macOS screencapture and stashes the
// result in the Drop Bar (or reveals it in Finder), Dropzone-4 style.
type Screenshot struct{}

func (Screenshot) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "screenshot",
		Name:        "Screenshot",
		Description: "Capture a screenshot and send it to the Drop Bar.",
		Icon:        "camera",
		Category:    "Capture",
		Events:      []string{model.EventClicked},
		Multi:       true,
		KeyModifier: "option",
		Options: []model.OptionField{
			{Key: "mode", Label: "Capture", Type: "select", Choices: []string{"interactive", "window", "screen"}, Default: "interactive"},
			{Key: "folder", Label: "Save to", Type: "folder", Placeholder: "~/Screenshots"},
			{Key: "after", Label: "After capture", Type: "select", Choices: []string{"dropbar", "reveal"}, Default: "dropbar"},
		},
	}
}

func (Screenshot) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	mode := inv.Target.Option("mode", "interactive")

	dir := expandHome(inv.Target.Option("folder", ""))
	if dir == "" {
		dir = expandHome("~/Screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return actions.Result{}, fmt.Errorf("creating screenshots folder: %w", err)
	}

	name := "Screenshot " + screenshotNow().Format("2006-01-02 at 15.04.05") + ".png"
	dst := filepath.Join(dir, name)

	var args []string
	switch mode {
	case "window":
		args = []string{"-w", dst}
	case "screen":
		args = []string{dst}
	default: // interactive
		args = []string{"-i", dst}
	}

	if err := screenshotCmd(ctx, args...).Run(); err != nil {
		return actions.Result{}, fmt.Errorf("capturing screenshot: %w", err)
	}

	if _, err := os.Stat(dst); err != nil {
		// No file means the user cancelled the capture (e.g. pressed Esc).
		return actions.Result{Message: "Screenshot cancelled"}, nil
	}

	switch inv.Target.Option("after", "dropbar") {
	case "reveal":
		if err := inv.Services.Reveal(dst); err != nil {
			return actions.Result{}, fmt.Errorf("revealing screenshot: %w", err)
		}
	default:
		if inv.AddDropBar != nil {
			inv.AddDropBar([]string{dst})
		}
	}

	return actions.Result{Message: "Screenshot saved — " + name}, nil
}

// expandHome expands a leading "~/" (or bare "~") to the user's home
// directory. Paths without that prefix are returned unchanged.
func expandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~/"))
		}
	}
	return p
}
