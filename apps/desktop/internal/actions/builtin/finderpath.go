package builtin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// FinderPath copies the full POSIX path of dropped files to the clipboard.
type FinderPath struct{}

func (FinderPath) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "finder-path",
		Name:        "Copy Path",
		Description: "Copy the full path of dropped files to the clipboard.",
		Icon:        "route",
		Category:    "File Management",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       false,
	}
}

func (FinderPath) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	if len(inv.Payload.Paths) == 0 {
		return actions.Result{}, errors.New("nothing to copy")
	}
	if err := inv.Services.CopyToClipboard(strings.Join(inv.Payload.Paths, "\n")); err != nil {
		return actions.Result{}, fmt.Errorf("copying path: %w", err)
	}
	return actions.Result{Message: fmt.Sprintf("Copied %d path(s)", len(inv.Payload.Paths))}, nil
}
