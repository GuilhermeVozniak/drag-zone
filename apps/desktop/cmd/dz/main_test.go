package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// capture runs fn with os.Stdout redirected and returns what it printed.
func capture(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	defer func() { os.Stdout = orig }()
	os.Stdout = w
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrintResultListTable(t *testing.T) {
	data := json.RawMessage(`[{"label":"Desktop","action":"folder","events":"dragged"}]`)
	out := capture(t, func() { printResult("list", data, false) })
	if !strings.Contains(out, "Desktop") || !strings.Contains(out, "folder") {
		t.Errorf("list output = %q", out)
	}
}

func TestPrintResultListItems(t *testing.T) {
	data := json.RawMessage(`[{"label":"a.txt","kind":"files","locked":true}]`)
	out := capture(t, func() { printResult("list-items", data, false) })
	if !strings.Contains(out, "1. a.txt (files) [locked]") {
		t.Errorf("list-items output = %q", out)
	}
}

func TestPrintResultJSONPassthrough(t *testing.T) {
	data := json.RawMessage(`{"any":"thing"}`)
	out := capture(t, func() { printResult("list", data, true) })
	if strings.TrimSpace(out) != `{"any":"thing"}` {
		t.Errorf("json passthrough = %q", out)
	}
}

func TestPrintResultStringAndNull(t *testing.T) {
	if out := capture(t, func() { printResult("add", json.RawMessage(`"added 2 item(s)"`), false) }); strings.TrimSpace(out) != "added 2 item(s)" {
		t.Errorf("string result = %q", out)
	}
	if out := capture(t, func() { printResult("clear", json.RawMessage(`null`), false) }); strings.TrimSpace(out) != "" {
		t.Errorf("null result should print nothing, got %q", out)
	}
}
