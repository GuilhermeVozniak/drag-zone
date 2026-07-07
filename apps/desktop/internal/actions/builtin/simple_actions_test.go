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

func TestTrashDelegatesToService(t *testing.T) {
	svc := &recServices{}
	res, err := Trash{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b"}}, svc))
	if err != nil || len(svc.Trashed) != 1 || len(svc.Trashed[0]) != 2 {
		t.Fatalf("trashed=%v err=%v", svc.Trashed, err)
	}
	if res.Message != "Moved 2 item(s) to Trash" {
		t.Errorf("message = %q", res.Message)
	}
}

func TestAirDropDelegatesToService(t *testing.T) {
	svc := &recServices{}
	res, err := AirDrop{}.Dropped(context.Background(),
		inv(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}}, svc))
	if err != nil || len(svc.AirDropped) != 1 || res.Message != "Sharing 1 item(s) via AirDrop" {
		t.Fatalf("airdropped=%v res=%+v err=%v", svc.AirDropped, res, err)
	}
}
