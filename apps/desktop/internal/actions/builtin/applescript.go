package builtin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// osascriptCmd is a seam so tests stub AppleScript execution (osascript would
// otherwise run a real, possibly UI-visible, script).
var osascriptCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "osascript", args...)
}

// RunAppleScript runs a user-configured AppleScript, passing dropped file
// paths (or dropped text) as `on run argv` arguments.
type RunAppleScript struct{}

func (RunAppleScript) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "run-applescript",
		Name:        "Run AppleScript",
		Description: "Run a custom AppleScript against dropped files or text.",
		Icon:        "scroll-text",
		Category:    "System",
		Events:      []string{model.EventDragged, model.EventClicked},
		Accepts:     []model.ItemKind{model.ItemFiles, model.ItemText},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "script", Label: "AppleScript", Type: "text", Placeholder: "on run argv\n\t-- ...\nend run"},
		},
	}
}

func (a RunAppleScript) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return a.run(ctx, inv)
}

func (a RunAppleScript) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return a.run(ctx, inv)
}

func (RunAppleScript) run(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	script := strings.TrimSpace(inv.Target.Option("script", ""))
	if script == "" {
		return actions.Result{}, errors.New("no script configured")
	}

	// The user's `on run argv` receives the dropped file paths, or the
	// dropped text as a single argument.
	var args []string
	if inv.Payload.Kind == model.ItemText {
		if t := inv.Payload.Text; t != "" {
			args = []string{t}
		}
	} else {
		args = append(args, inv.Payload.Paths...)
	}

	cmd := osascriptCmd(ctx, append([]string{"-e", script}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return actions.Result{}, fmt.Errorf("running AppleScript: %w (%s)", err, strings.TrimSpace(out.String()))
	}

	msg := "AppleScript ran"
	if trimmed := strings.TrimSpace(out.String()); trimmed != "" {
		msg = trimmed
	}
	return actions.Result{Message: msg}, nil
}
