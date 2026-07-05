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

// OpenApplication launches an app; dropped files are opened with that app.
type OpenApplication struct{}

func (OpenApplication) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "open-app",
		Name:        "Open Application",
		Description: "Launch an application. Dropped files are opened with it.",
		Icon:        "app-window",
		Category:    "Applications",
		Events:      []string{model.EventDragged, model.EventClicked},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "app", Label: "Application", Type: "app", Required: true},
		},
	}
}

func (OpenApplication) Clicked(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	return actions.Result{}, openWith(inv.Target.Option("app", ""), nil)
}

func (OpenApplication) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	app := inv.Target.Option("app", "")
	if err := openWith(app, inv.Payload.Paths); err != nil {
		return actions.Result{}, err
	}
	return actions.Result{Message: fmt.Sprintf("Opened %d item(s) with %s", len(inv.Payload.Paths), appName(app))}, nil
}

func openWith(app string, paths []string) error {
	if app == "" {
		return fmt.Errorf("no application configured")
	}
	args := append([]string{"-a", app}, paths...)
	out, err := exec.Command("open", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("open %s: %s", appName(app), strings.TrimSpace(string(out)))
	}
	return nil
}

func appName(app string) string {
	return strings.TrimSuffix(filepath.Base(app), ".app")
}
