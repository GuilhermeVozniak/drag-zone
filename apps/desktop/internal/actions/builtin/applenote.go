package builtin

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// noteCmd is a seam so tests stub note creation (osascript would otherwise
// create a real, visible Apple Note).
var noteCmd = func(ctx context.Context, script string) *exec.Cmd {
	return exec.CommandContext(ctx, "osascript", "-e", script)
}

// AppleNote creates a note in Apple Notes from dropped text or file names.
type AppleNote struct{}

func (AppleNote) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "apple-note",
		Name:        "Create Apple Note",
		Description: "Create a note in Apple Notes from dropped text or files.",
		Icon:        "notebook-pen",
		Category:    "System",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemText, model.ItemFiles},
		Multi:       false,
	}
}

func (AppleNote) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	var lines []string
	if inv.Payload.Kind == model.ItemFiles {
		for _, p := range inv.Payload.Paths {
			lines = append(lines, filepath.Base(p))
		}
	} else if t := strings.TrimSpace(inv.Payload.Text); t != "" {
		lines = strings.Split(inv.Payload.Text, "\n")
	}
	if len(lines) == 0 {
		return actions.Result{}, errors.New("nothing to add to note")
	}

	body := noteHTMLBody(lines)
	script := fmt.Sprintf(`tell application "Notes" to make new note with properties {body:%s}`, appleScriptString(body))
	if err := noteCmd(ctx, script).Run(); err != nil {
		return actions.Result{}, fmt.Errorf("creating note: %w", err)
	}
	return actions.Result{Message: "Note created"}, nil
}

// noteHTMLBody joins lines into an HTML note body, escaping the HTML-unsafe
// characters in each line and separating them with <br> so Notes renders
// them one per line.
func noteHTMLBody(lines []string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	escaped := make([]string, len(lines))
	for i, l := range lines {
		escaped[i] = replacer.Replace(l)
	}
	return strings.Join(escaped, "<br>")
}

// appleScriptString quotes s as an AppleScript string literal, escaping
// backslashes and double quotes so embedded content can't break out of the
// literal (or inject additional script).
func appleScriptString(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(s) + `"`
}
