package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// PrintFiles sends dropped documents to the default printer.
type PrintFiles struct{}

func (PrintFiles) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "print",
		Name:        "Print",
		Description: "Print dropped documents on the default printer.",
		Icon:        "printer",
		Category:    "Utilities",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (PrintFiles) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	for _, path := range inv.Payload.Paths {
		inv.Progress.Detail(filepath.Base(path))
		out, err := exec.CommandContext(ctx, "lp", path).CombinedOutput()
		if err != nil {
			return actions.Result{}, fmt.Errorf("printing %s: %s", filepath.Base(path), strings.TrimSpace(string(out)))
		}
	}
	return actions.Result{Message: fmt.Sprintf("Sent %d document(s) to printer", len(inv.Payload.Paths))}, nil
}
