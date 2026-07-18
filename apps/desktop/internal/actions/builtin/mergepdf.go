package builtin

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// MergePDFs concatenates dropped PDFs into a single document, placed in the
// Drop Bar.
type MergePDFs struct{}

func (MergePDFs) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "merge-pdfs",
		Name:        "Merge PDFs",
		Description: "Merge dropped PDFs into one document (placed in the Drop Bar).",
		Icon:        "file-stack",
		Category:    "File Management",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (MergePDFs) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	var pdfs []string
	for _, p := range inv.Payload.Paths {
		if strings.EqualFold(filepath.Ext(p), ".pdf") {
			pdfs = append(pdfs, p)
		}
	}
	if len(pdfs) < 2 {
		return actions.Result{}, fmt.Errorf("merge-pdfs: need at least two PDFs")
	}

	dir := filepath.Dir(pdfs[0])
	out := fsutil.UniqueDest(dir, "Merged.pdf")

	if err := api.MergeCreateFile(pdfs, out, false, nil); err != nil {
		return actions.Result{}, fmt.Errorf("merge-pdfs: %w", err)
	}

	if inv.AddDropBar != nil {
		inv.AddDropBar([]string{out})
	}

	return actions.Result{Message: fmt.Sprintf("Merged %d PDFs", len(pdfs))}, nil
}
