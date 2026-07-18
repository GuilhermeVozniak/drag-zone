package builtin

import (
	"context"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestPrintSpec(t *testing.T) {
	spec := PrintFiles{}.Spec()
	if got := spec.Events; len(got) != 1 || got[0] != model.EventDragged {
		t.Errorf("events = %v, want [dragged]", got)
	}
	if got := spec.Accepts; len(got) != 1 || got[0] != model.ItemFiles {
		t.Errorf("accepts = %v, want [files]", got)
	}
}

// An empty payload never reaches exec.Command("lp", ...): the loop over
// Paths has nothing to iterate, so this is safe to run without a real
// printer or `lp` binary.
func TestPrintDroppedEmptyPayloadSkipsLp(t *testing.T) {
	i := actions.Invocation{
		Target:   model.Target{Label: "t"},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	}
	res, err := (PrintFiles{}).Dropped(context.Background(), i)
	if err != nil {
		t.Fatalf("empty payload should not error, got %v", err)
	}
	if res.Message != "Sent 0 document(s) to printer" {
		t.Errorf("message = %q", res.Message)
	}
}
