package builtin

import (
	"context"
	"errors"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func inv(p model.Payload, svc actions.Services) actions.Invocation {
	return actions.Invocation{
		Target:   model.Target{Label: "t"},
		Payload:  p,
		Progress: nullProgress{},
		Services: svc,
	}
}

func TestClipboardCopiesTextAndPaths(t *testing.T) {
	svc := &recServices{}
	res, err := CopyToClipboard{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemText, Text: "hello"}, svc))
	if err != nil || svc.Clipboard != "hello" || res.Message != "Copied to clipboard" {
		t.Fatalf("text: clip=%q res=%+v err=%v", svc.Clipboard, res, err)
	}
	svc = &recServices{}
	_, err = CopyToClipboard{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b"}}, svc))
	if err != nil || svc.Clipboard != "/a\n/b" {
		t.Fatalf("files: clip=%q err=%v", svc.Clipboard, err)
	}
}

func TestClipboardSurfacesServiceError(t *testing.T) {
	svc := &recServices{ClipboardErr: errors.New("no clip")}
	_, err := CopyToClipboard{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemText, Text: "x"}, svc))
	if err == nil {
		t.Error("clipboard error should propagate")
	}
}

func TestClipboardSpec(t *testing.T) {
	spec := CopyToClipboard{}.Spec()
	if got := spec.Events; len(got) != 1 || got[0] != model.EventDragged {
		t.Errorf("events = %v, want [dragged]", got)
	}
	want := []model.ItemKind{model.ItemFiles, model.ItemText, model.ItemURL}
	if len(spec.Accepts) != len(want) {
		t.Fatalf("accepts = %v, want %v", spec.Accepts, want)
	}
	for i, k := range want {
		if spec.Accepts[i] != k {
			t.Errorf("accepts[%d] = %q, want %q", i, spec.Accepts[i], k)
		}
	}
}

func TestTrashDelegatesToService(t *testing.T) {
	svc := &recServices{}
	res, err := Trash{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b"}}, svc))
	if err != nil || len(svc.Trashed) != 1 || len(svc.Trashed[0]) != 2 {
		t.Fatalf("trashed=%v err=%v", svc.Trashed, err)
	}
	if svc.Trashed[0][0] != "/a" || svc.Trashed[0][1] != "/b" {
		t.Errorf("trashed paths = %v, want [/a /b]", svc.Trashed[0])
	}
	if res.Message != "Moved 2 item(s) to Trash" {
		t.Errorf("message = %q", res.Message)
	}
}

func TestTrashSurfacesServiceError(t *testing.T) {
	svc := &recServices{TrashErr: errors.New("trash denied")}
	_, err := Trash{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}}, svc))
	if err == nil {
		t.Error("trash error should propagate")
	}
	if len(svc.Trashed) != 0 {
		t.Errorf("trashed should stay empty on error, got %v", svc.Trashed)
	}
}

func TestTrashSpec(t *testing.T) {
	spec := Trash{}.Spec()
	if got := spec.Events; len(got) != 1 || got[0] != model.EventDragged {
		t.Errorf("events = %v, want [dragged]", got)
	}
	if got := spec.Accepts; len(got) != 1 || got[0] != model.ItemFiles {
		t.Errorf("accepts = %v, want [files]", got)
	}
}

func TestAirDropDelegatesToService(t *testing.T) {
	svc := &recServices{}
	res, err := AirDrop{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}}, svc))
	if err != nil || len(svc.AirDropped) != 1 || res.Message != "Sharing 1 item(s) via AirDrop" {
		t.Fatalf("airdropped=%v res=%+v err=%v", svc.AirDropped, res, err)
	}
	if svc.AirDropped[0][0] != "/a" {
		t.Errorf("airdropped paths = %v, want [/a]", svc.AirDropped[0])
	}
}

func TestAirDropSurfacesServiceError(t *testing.T) {
	svc := &recServices{AirDropErr: errors.New("no nearby devices")}
	_, err := AirDrop{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}}, svc))
	if err == nil {
		t.Error("airdrop error should propagate")
	}
	if len(svc.AirDropped) != 0 {
		t.Errorf("airdropped should stay empty on error, got %v", svc.AirDropped)
	}
}

func TestAirDropSpec(t *testing.T) {
	spec := AirDrop{}.Spec()
	if got := spec.Events; len(got) != 1 || got[0] != model.EventDragged {
		t.Errorf("events = %v, want [dragged]", got)
	}
	if got := spec.Accepts; len(got) != 1 || got[0] != model.ItemFiles {
		t.Errorf("accepts = %v, want [files]", got)
	}
}
