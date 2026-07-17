package main

import "testing"

func TestEmitUsesOnEmitHook(t *testing.T) {
	app := newTestApp(t)
	var got []string
	app.onEmit = func(event string, _ ...any) { got = append(got, event) }
	app.emit("x:changed", 1)
	if len(got) != 1 || got[0] != "x:changed" {
		t.Fatalf("onEmit not invoked, got %v", got)
	}
}
