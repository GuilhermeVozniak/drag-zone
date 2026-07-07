package model

import "testing"

func TestPayloadHasModifier(t *testing.T) {
	p := Payload{Modifiers: []string{"Option", "Shift"}}
	if !p.HasModifier("Option") {
		t.Error("HasModifier(Option) = false, want true")
	}
	if p.HasModifier("Command") {
		t.Error("HasModifier(Command) = true, want false")
	}
	if (Payload{}).HasModifier("Option") {
		t.Error("empty payload should have no modifiers")
	}
}

func TestPayloadIsEmpty(t *testing.T) {
	cases := []struct {
		name string
		p    Payload
		want bool
	}{
		{"nothing", Payload{}, true},
		{"paths", Payload{Paths: []string{"/a"}}, false},
		{"text", Payload{Text: "hi"}, false},
		{"empty slice", Payload{Paths: []string{}}, true},
	}
	for _, c := range cases {
		if got := c.p.IsEmpty(); got != c.want {
			t.Errorf("%s: IsEmpty() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTargetOption(t *testing.T) {
	t1 := Target{Options: map[string]string{"mode": "copy", "blank": ""}}
	if got := t1.Option("mode", "move"); got != "copy" {
		t.Errorf(`Option("mode") = %q, want "copy"`, got)
	}
	// An empty stored value falls through to the default.
	if got := t1.Option("blank", "fallback"); got != "fallback" {
		t.Errorf(`Option("blank") = %q, want "fallback"`, got)
	}
	if got := t1.Option("missing", "def"); got != "def" {
		t.Errorf(`Option("missing") = %q, want "def"`, got)
	}
	// Nil map must not panic.
	if got := (Target{}).Option("x", "d"); got != "d" {
		t.Errorf("nil-map Option = %q, want d", got)
	}
}
