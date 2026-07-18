package builtin

import (
	"context"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// RemoveImageMetadata wraps platform.StripImageMetadata, a real cgo call into
// the native ImageIO bridge. Per the task brief, these tests deliberately
// never invoke that bridge (not even on an error path) — they cover only
// Spec metadata and the empty-payload branch, which returns before the loop
// (and therefore the cgo call) ever runs.

func TestMetadataSpec(t *testing.T) {
	spec := RemoveImageMetadata{}.Spec()
	if spec.ID != "remove-metadata" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %+v", spec.Accepts)
	}
	if len(spec.Options) != 0 {
		t.Errorf("Options = %+v, want none", spec.Options)
	}
}

func TestMetadataDroppedEmptyPayload(t *testing.T) {
	res, err := RemoveImageMetadata{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "Cleaned 0 image(s)" {
		t.Errorf("message = %q", res.Message)
	}
}
