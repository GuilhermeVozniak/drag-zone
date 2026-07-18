package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// writeMinimalPDF writes a tiny, valid single-page PDF (with a correct xref
// table) to path — enough for pdfcpu to read, merge, and page-count.
func writeMinimalPDF(t *testing.T, path, text string) {
	t.Helper()

	var buf bytes.Buffer
	var offsets [6]int

	buf.WriteString("%PDF-1.4\n")

	writeObj := func(n int, body string) {
		offsets[n] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}

	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
		"/Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	content := fmt.Sprintf("BT /F1 24 Tf 72 700 Td (%s) Tj ET", text)
	writeObj(5, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))

	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString("0 6\n")
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	buf.WriteString("trailer\n")
	buf.WriteString("<< /Size 6 /Root 1 0 R >>\n")
	buf.WriteString("startxref\n")
	fmt.Fprintf(&buf, "%d\n", xrefOffset)
	buf.WriteString("%%EOF")

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMergePDFsSpec(t *testing.T) {
	spec := MergePDFs{}.Spec()
	if spec.ID != "merge-pdfs" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %+v", spec.Accepts)
	}
}

func TestMergePDFsDropped(t *testing.T) {
	dir := t.TempDir()
	pdf1 := filepath.Join(dir, "a.pdf")
	pdf2 := filepath.Join(dir, "b.pdf")
	writeMinimalPDF(t, pdf1, "Page A")
	writeMinimalPDF(t, pdf2, "Page B")

	// Sanity-check the fixtures are individually valid single-page PDFs
	// before relying on them to exercise the merge path.
	for _, p := range []string{pdf1, pdf2} {
		if n, err := api.PageCountFile(p); err != nil || n != 1 {
			t.Fatalf("fixture %s: PageCountFile = %d, %v, want 1, nil", p, n, err)
		}
	}

	var addedToDropBar [][]string
	res, err := MergePDFs{}.Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{},
		Payload: model.Payload{Kind: model.ItemFiles, Paths: []string{pdf1, pdf2}},
		AddDropBar: func(paths []string) {
			addedToDropBar = append(addedToDropBar, paths)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Merged 2 PDFs" {
		t.Errorf("message = %q", res.Message)
	}

	out := filepath.Join(dir, "Merged.pdf")
	n, err := api.PageCountFile(out)
	if err != nil {
		t.Fatalf("PageCountFile(%s): %v", out, err)
	}
	if n != 2 {
		t.Errorf("merged page count = %d, want 2", n)
	}

	if len(addedToDropBar) != 1 || len(addedToDropBar[0]) != 1 || addedToDropBar[0][0] != out {
		t.Errorf("AddDropBar got %+v, want [[%s]]", addedToDropBar, out)
	}
}

func TestMergePDFsDroppedRequiresAtLeastTwo(t *testing.T) {
	dir := t.TempDir()
	pdf1 := filepath.Join(dir, "a.pdf")
	writeMinimalPDF(t, pdf1, "Page A")

	if _, err := (MergePDFs{}).Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{},
		Payload: model.Payload{Kind: model.ItemFiles, Paths: []string{pdf1}},
	}); err == nil {
		t.Error("expected an error for fewer than two PDFs")
	}
}

func TestMergePDFsDroppedEmptyPayload(t *testing.T) {
	if _, err := (MergePDFs{}).Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{},
		Payload: model.Payload{Kind: model.ItemFiles},
	}); err == nil {
		t.Error("expected an error for an empty payload")
	}
}

func TestMergePDFsDroppedFiltersNonPDFs(t *testing.T) {
	dir := t.TempDir()
	pdf1 := filepath.Join(dir, "a.pdf")
	writeMinimalPDF(t, pdf1, "Page A")
	txt := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txt, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Only one real PDF among the dropped items — still below the minimum.
	if _, err := (MergePDFs{}).Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{},
		Payload: model.Payload{Kind: model.ItemFiles, Paths: []string{pdf1, txt}},
	}); err == nil {
		t.Error("expected an error when fewer than two PDFs are present")
	}
}
