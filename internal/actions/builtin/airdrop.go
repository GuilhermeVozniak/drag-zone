package builtin

import (
	"context"
	"fmt"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// AirDrop shares dropped files via AirDrop.
type AirDrop struct{}

func (AirDrop) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "airdrop",
		Name:        "AirDrop",
		Description: "Share dropped files with nearby devices via AirDrop.",
		Icon:        "wifi",
		Category:    "Sharing",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (AirDrop) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	if err := inv.Services.AirDrop(inv.Payload.Paths); err != nil {
		return actions.Result{}, err
	}
	return actions.Result{Message: fmt.Sprintf("Sharing %d item(s) via AirDrop", len(inv.Payload.Paths))}, nil
}
