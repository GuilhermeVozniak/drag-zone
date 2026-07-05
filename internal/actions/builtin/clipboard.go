package builtin

import (
	"context"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// CopyToClipboard puts dropped text (or file paths) on the clipboard.
type CopyToClipboard struct{}

func (CopyToClipboard) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "copy-to-clipboard",
		Name:        "Copy to Clipboard",
		Description: "Copy dropped text, URLs, or file paths to the clipboard.",
		Icon:        "clipboard-copy",
		Category:    "Utilities",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles, model.ItemText, model.ItemURL},
	}
}

func (CopyToClipboard) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	text := inv.Payload.Text
	if inv.Payload.Kind == model.ItemFiles {
		text = strings.Join(inv.Payload.Paths, "\n")
	}
	if err := inv.Services.CopyToClipboard(text); err != nil {
		return actions.Result{}, err
	}
	return actions.Result{Message: "Copied to clipboard"}, nil
}
