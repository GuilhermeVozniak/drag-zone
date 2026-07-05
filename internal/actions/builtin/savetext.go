package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// SaveText saves dropped text snippets or URLs into a configured folder.
type SaveText struct{}

func (SaveText) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "save-text",
		Name:        "Save Text",
		Description: "Save dropped text snippets as files in a folder.",
		Icon:        "type",
		Category:    "Utilities",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemText, model.ItemURL},
		Options: []model.OptionField{
			{Key: "path", Label: "Save to folder", Type: "folder", Required: true},
		},
	}
}

func (SaveText) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	dir := inv.Target.Option("path", "")
	if dir == "" {
		return actions.Result{}, fmt.Errorf("no folder configured")
	}
	text := inv.Payload.Text
	if strings.TrimSpace(text) == "" {
		return actions.Result{}, fmt.Errorf("nothing to save")
	}
	dst := fsutil.UniqueDest(dir, snippetName(text))
	if err := os.WriteFile(dst, []byte(text), 0o644); err != nil {
		return actions.Result{}, fmt.Errorf("saving text: %w", err)
	}
	return actions.Result{Message: "Saved " + filepath.Base(dst)}, nil
}

// snippetName derives a readable file name from the snippet's first words.
func snippetName(text string) string {
	fields := strings.Fields(text)
	var words []string
	var length int
	for _, f := range fields {
		f = strings.Map(func(r rune) rune {
			if strings.ContainsRune(`/\:*?"<>|`, r) {
				return -1
			}
			return r
		}, f)
		if f == "" {
			continue
		}
		words = append(words, f)
		length += len(f)
		if len(words) >= 6 || length > 40 {
			break
		}
	}
	if len(words) == 0 {
		return "Snippet " + time.Now().Format("2006-01-02 15.04.05") + ".txt"
	}
	return strings.Join(words, " ") + ".txt"
}
