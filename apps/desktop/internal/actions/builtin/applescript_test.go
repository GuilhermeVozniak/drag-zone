package builtin

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// withFakeOsascriptCmd swaps osascriptCmd for a fake that records the args it
// was invoked with, running printFn (or a no-op "true") so no real osascript
// runs. Restores the original osascriptCmd on cleanup.
func withFakeOsascriptCmd(t *testing.T, bin string, binArgs ...string) *[][]string {
	t.Helper()
	var calls [][]string
	orig := osascriptCmd
	t.Cleanup(func() { osascriptCmd = orig })
	osascriptCmd = func(ctx context.Context, args ...string) *exec.Cmd {
		calls = append(calls, append([]string(nil), args...))
		if bin == "" {
			return exec.CommandContext(ctx, "true")
		}
		return exec.CommandContext(ctx, bin, binArgs...)
	}
	return &calls
}

func TestRunAppleScriptSpec(t *testing.T) {
	spec := RunAppleScript{}.Spec()
	if spec.ID != "run-applescript" {
		t.Errorf("ID = %q, want %q", spec.ID, "run-applescript")
	}
	if spec.Name != "Run AppleScript" {
		t.Errorf("Name = %q, want %q", spec.Name, "Run AppleScript")
	}
	if spec.Icon != "scroll-text" {
		t.Errorf("Icon = %q, want %q", spec.Icon, "scroll-text")
	}
	if spec.Category != "System" {
		t.Errorf("Category = %q, want %q", spec.Category, "System")
	}
	wantEvents := []string{model.EventDragged, model.EventClicked}
	if len(spec.Events) != len(wantEvents) || spec.Events[0] != wantEvents[0] || spec.Events[1] != wantEvents[1] {
		t.Errorf("Events = %v, want %v", spec.Events, wantEvents)
	}
	wantAccepts := []model.ItemKind{model.ItemFiles, model.ItemText}
	if len(spec.Accepts) != len(wantAccepts) || spec.Accepts[0] != wantAccepts[0] || spec.Accepts[1] != wantAccepts[1] {
		t.Errorf("Accepts = %v, want %v", spec.Accepts, wantAccepts)
	}
	if !spec.Multi {
		t.Error("expected Multi = true")
	}
	if len(spec.Options) != 1 || spec.Options[0].Key != "script" {
		t.Errorf("Options = %+v, want a single 'script' option", spec.Options)
	}
}

func TestRunAppleScriptDroppedFilesPassedAsArgv(t *testing.T) {
	calls := withFakeOsascriptCmd(t, "")
	res, err := RunAppleScript{}.Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{Options: map[string]string{"script": "on run argv\nend run"}},
		Payload: model.Payload{Kind: model.ItemFiles, Paths: []string{"/a/one.txt", "/b/two.txt"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "AppleScript ran" {
		t.Errorf("Message = %q, want %q", res.Message, "AppleScript ran")
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 osascript call, got %d: %v", len(*calls), *calls)
	}
	args := (*calls)[0]
	want := []string{"-e", "on run argv\nend run", "/a/one.txt", "/b/two.txt"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q (full: %v)", i, args[i], want[i], args)
		}
	}
}

func TestRunAppleScriptDroppedTextPassedAsSingleArg(t *testing.T) {
	calls := withFakeOsascriptCmd(t, "")
	_, err := RunAppleScript{}.Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{Options: map[string]string{"script": "on run argv\nend run"}},
		Payload: model.Payload{Kind: model.ItemText, Text: "hello world"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := (*calls)[0]
	want := []string{"-e", "on run argv\nend run", "hello world"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestRunAppleScriptClickedNoPayloadRunsWithNoArgs(t *testing.T) {
	calls := withFakeOsascriptCmd(t, "")
	_, err := RunAppleScript{}.Clicked(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{"script": "beep"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := (*calls)[0]
	want := []string{"-e", "beep"}
	if len(args) != len(want) || args[0] != want[0] || args[1] != want[1] {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestRunAppleScriptBlankScriptErrorsBeforeExec(t *testing.T) {
	calls := withFakeOsascriptCmd(t, "")
	_, err := RunAppleScript{}.Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{Options: map[string]string{"script": "   "}},
		Payload: model.Payload{Kind: model.ItemFiles, Paths: []string{"/a.txt"}},
	})
	if err == nil {
		t.Fatal("expected an error for a blank script")
	}
	if err.Error() != "no script configured" {
		t.Errorf("error = %q, want %q", err.Error(), "no script configured")
	}
	if len(*calls) != 0 {
		t.Errorf("osascript should not be invoked with a blank script, got %v", *calls)
	}
}

func TestRunAppleScriptFailureWrapped(t *testing.T) {
	withFakeOsascriptCmd(t, "false")
	_, err := RunAppleScript{}.Dropped(context.Background(), actions.Invocation{
		Target:  model.Target{Options: map[string]string{"script": "error \"boom\""}},
		Payload: model.Payload{Kind: model.ItemText, Text: "x"},
	})
	if err == nil {
		t.Fatal("expected an error when osascript fails")
	}
	if !strings.Contains(err.Error(), "running AppleScript") {
		t.Errorf("error = %q, want it to mention %q", err.Error(), "running AppleScript")
	}
}
