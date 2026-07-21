package builtin

import (
	"context"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// CopyToClipboard puts dropped text (or whole files) on the clipboard.
type CopyToClipboard struct{}

func (CopyToClipboard) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "copy-to-clipboard",
		Name:        "Copy to Clipboard",
		Description: "Copy dropped text, URLs, or files to the clipboard.",
		Icon:        "clipboard-copy",
		Category:    "Utilities",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles, model.ItemText, model.ItemURL},
	}
}

func (CopyToClipboard) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	// Files go on the pasteboard as file URLs so they can be pasted as
	// files into Finder and other apps, not as path strings.
	if inv.Payload.Kind == model.ItemFiles && len(inv.Payload.Paths) > 0 {
		if err := inv.Services.CopyFilesToClipboard(inv.Payload.Paths); err != nil {
			return actions.Result{}, err
		}
		return actions.Result{Message: "Copied to clipboard"}, nil
	}
	if err := inv.Services.CopyToClipboard(inv.Payload.Text); err != nil {
		return actions.Result{}, err
	}
	return actions.Result{Message: "Copied to clipboard"}, nil
}
