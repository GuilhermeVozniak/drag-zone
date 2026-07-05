package builtin

import (
	"context"
	"fmt"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// Trash moves dropped files to the Finder trash.
type Trash struct{}

func (Trash) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "trash",
		Name:        "Move to Trash",
		Description: "Move dropped files to the Trash.",
		Icon:        "trash-2",
		Category:    "File Management",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (Trash) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	if err := inv.Services.Trash(inv.Payload.Paths); err != nil {
		return actions.Result{}, err
	}
	return actions.Result{Message: fmt.Sprintf("Moved %d item(s) to Trash", len(inv.Payload.Paths))}, nil
}
