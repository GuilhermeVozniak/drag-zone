package builtin

import (
	"context"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestOpenApplicationSpec(t *testing.T) {
	spec := OpenApplication{}.Spec()
	wantEvents := []string{model.EventDragged, model.EventClicked}
	if len(spec.Events) != len(wantEvents) {
		t.Fatalf("events = %v, want %v", spec.Events, wantEvents)
	}
	for i, e := range wantEvents {
		if spec.Events[i] != e {
			t.Errorf("events[%d] = %q, want %q", i, spec.Events[i], e)
		}
	}
	if got := spec.Accepts; len(got) != 1 || got[0] != model.ItemFiles {
		t.Errorf("accepts = %v, want [files]", got)
	}
	if !spec.Multi {
		t.Error("open-app should be Multi")
	}
	if len(spec.Options) != 1 || spec.Options[0].Key != "app" || !spec.Options[0].Required {
		t.Errorf("options = %+v, want one required %q option", spec.Options, "app")
	}
}

// Clicked and Dropped both reject a target with no configured application
// before ever shelling out to `open`.
func TestOpenApplicationClickedNoAppConfigured(t *testing.T) {
	i := actions.Invocation{
		Target:   model.Target{Label: "t"},
		Progress: nullProgress{},
	}
	if _, err := (OpenApplication{}).Clicked(context.Background(), i); err == nil {
		t.Error("Clicked with no app option should error")
	}
}

func TestOpenApplicationDroppedNoAppConfigured(t *testing.T) {
	i := actions.Invocation{
		Target:   model.Target{Label: "t"},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}},
		Progress: nullProgress{},
	}
	if _, err := (OpenApplication{}).Dropped(context.Background(), i); err == nil {
		t.Error("Dropped with no app option should error")
	}
}

func TestOpenWithNoApp(t *testing.T) {
	if err := openWith("", nil); err == nil {
		t.Error("openWith with empty app should error without shelling out")
	}
}
