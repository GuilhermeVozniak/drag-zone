package builtin

import (
	"context"
	"errors"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestFinderPathSpec(t *testing.T) {
	spec := FinderPath{}.Spec()
	if spec.ID != "finder-path" {
		t.Errorf("ID = %q, want %q", spec.ID, "finder-path")
	}
	if spec.Name != "Copy Path" {
		t.Errorf("Name = %q, want %q", spec.Name, "Copy Path")
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventDragged {
		t.Errorf("Events = %v, want [%s]", spec.Events, model.EventDragged)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %v, want [%s]", spec.Accepts, model.ItemFiles)
	}
	if spec.Multi {
		t.Error("expected Multi = false")
	}
	if spec.Icon != "route" {
		t.Errorf("Icon = %q, want %q", spec.Icon, "route")
	}
}

func TestFinderPathCopiesJoinedPaths(t *testing.T) {
	svc := &recServices{}
	res, err := FinderPath{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/a/one.txt", "/b/two.txt"}},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Clipboard != "/a/one.txt\n/b/two.txt" {
		t.Errorf("clipboard = %q, want joined paths", svc.Clipboard)
	}
	if res.Message != "Copied 2 path(s)" {
		t.Errorf("message = %q, want %q", res.Message, "Copied 2 path(s)")
	}
}

func TestFinderPathEmptyPayloadErrors(t *testing.T) {
	svc := &recServices{}
	_, err := FinderPath{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles},
		Services: svc,
	})
	if err == nil {
		t.Error("expected an error for an empty payload")
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard should stay untouched, got %q", svc.Clipboard)
	}
}

func TestFinderPathSurfacesServiceError(t *testing.T) {
	svc := &recServices{ClipboardErr: errors.New("clip locked")}
	_, err := FinderPath{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}},
		Services: svc,
	})
	if err == nil {
		t.Error("expected the clipboard error to propagate")
	}
	if !errors.Is(err, svc.ClipboardErr) {
		t.Errorf("error should wrap the clipboard error, got %v", err)
	}
}
