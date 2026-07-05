package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// Folder copies or moves dropped files into a chosen folder; clicking opens
// the folder in Finder. Text drops are saved as a text clipping file.
type Folder struct{}

func (Folder) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "folder",
		Name:        "Folder",
		Description: "Copy or move dropped files to a folder. Click to open it in Finder.",
		Icon:        "folder",
		Category:    "File Management",
		Events:      []string{model.EventDragged, model.EventClicked},
		Accepts:     []model.ItemKind{model.ItemFiles, model.ItemText, model.ItemURL},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "path", Label: "Folder", Type: "folder", Required: true},
			{Key: "mode", Label: "On drop", Type: "select", Choices: []string{"copy", "move"}, Default: "copy"},
		},
	}
}

func (Folder) Clicked(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	return actions.Result{}, inv.Services.OpenPath(inv.Target.Option("path", ""))
}

func (Folder) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	dir := inv.Target.Option("path", "")
	if dir == "" {
		return actions.Result{}, fmt.Errorf("no folder configured")
	}

	if inv.Payload.Kind != model.ItemFiles {
		return saveTextClipping(dir, inv.Payload)
	}

	mode := inv.Target.Option("mode", "copy")
	// Holding Option inverts the configured behavior, like Dropzone.
	if inv.Payload.HasModifier("Option") {
		if mode == "copy" {
			mode = "move"
		} else {
			mode = "copy"
		}
	}
	total := fsutil.TotalSize(inv.Payload.Paths)
	var done atomic.Int64
	onBytes := func(n int64) {
		if total > 0 {
			inv.Progress.Percent(int(done.Add(n) * 100 / total))
		}
	}

	for _, src := range inv.Payload.Paths {
		inv.Progress.Detail(filepath.Base(src))
		var err error
		if mode == "move" {
			_, err = fsutil.MovePath(src, dir, onBytes)
		} else {
			_, err = fsutil.CopyPath(src, dir, onBytes)
		}
		if err != nil {
			return actions.Result{}, fmt.Errorf("%s %s: %w", mode, filepath.Base(src), err)
		}
	}

	verb := "Copied"
	if mode == "move" {
		verb = "Moved"
	}
	return actions.Result{Message: fmt.Sprintf("%s %d item(s) to %s", verb, len(inv.Payload.Paths), filepath.Base(dir))}, nil
}

func saveTextClipping(dir string, p model.Payload) (actions.Result, error) {
	name := "Dropped Text.txt"
	if p.Kind == model.ItemURL {
		name = "Dropped Link.url"
	}
	dst := fsutil.UniqueDest(dir, name)
	content := p.Text
	if p.Kind == model.ItemURL {
		content = "[InternetShortcut]\nURL=" + p.Text + "\n"
	}
	if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
		return actions.Result{}, err
	}
	return actions.Result{Message: "Saved " + filepath.Base(dst)}, nil
}
