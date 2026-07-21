package bundles

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// Host provides callbacks a running script can trigger in the app.
type Host struct {
	// SaveValue persists a value into the target's options.
	SaveValue func(targetID, name, value string)
	// RemoveValue removes a persisted option value.
	RemoveValue func(targetID, name string)
	// AddDropBar stashes file paths in the Drop Bar.
	AddDropBar func(paths []string)
	// RequestInput shows an input dialog and blocks until the user answers.
	// ok is false when the user cancelled or the task's context was cancelled.
	RequestInput func(ctx context.Context, title, prompt string) (value string, ok bool)
	// Console receives the script's raw output lines (plain stdout and
	// stderr) for the debug console. Optional.
	Console func(line string)
	// RunFailed is called when the script run ends in an error; the app uses
	// it to auto-open the debug console, like Dropzone. Optional.
	RunFailed func()
}

// ScriptAction adapts a .dzbundle to the actions.Action interface.
type ScriptAction struct {
	bundlePath  string
	scriptPath  string
	interpreter string // "ruby" or "python3"
	meta        Meta
	iconB64     string
	host        Host
}

func (s *ScriptAction) Spec() model.ActionSpec {
	icon := "file"
	if s.iconB64 != "" {
		icon = "data:image/png;base64," + s.iconB64
	}
	var accepts []model.ItemKind
	for _, h := range s.meta.Handles {
		switch h {
		case "Files":
			accepts = append(accepts, model.ItemFiles)
		case "Text":
			accepts = append(accepts, model.ItemText, model.ItemURL)
		}
	}
	var events []string
	for _, e := range s.meta.Events {
		switch e {
		case "Dragged":
			events = append(events, model.EventDragged)
		case "Clicked":
			events = append(events, model.EventClicked)
		}
	}
	return model.ActionSpec{
		ID:          "bundle:" + s.meta.UniqueID,
		Name:        s.meta.Name,
		Description: s.meta.Description,
		Icon:        icon,
		Category:    "Add-on Actions",
		Events:      events,
		Accepts:     accepts,
		Options:     optionFields(s.meta.OptionsNIB),
		Multi:       true,
	}
}

func (s *ScriptAction) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return s.run(ctx, inv, model.EventDragged)
}

func (s *ScriptAction) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return s.run(ctx, inv, model.EventClicked)
}

func (s *ScriptAction) run(ctx context.Context, inv actions.Invocation, event string) (actions.Result, error) {
	shim, shimName := rubyShim, "shim.rb"
	if s.interpreter == "python3" {
		shim, shimName = pythonShim, "shim.py"
	}
	tmp, err := os.MkdirTemp("", "dragzone-action-*")
	if err != nil {
		return actions.Result{}, err
	}
	defer os.RemoveAll(tmp)
	shimPath := filepath.Join(tmp, shimName)
	if err := os.WriteFile(shimPath, []byte(shim), 0o644); err != nil {
		return actions.Result{}, err
	}

	args := []string{shimPath}
	var draggedType string
	switch inv.Payload.Kind {
	case model.ItemFiles:
		args = append(args, inv.Payload.Paths...)
		draggedType = "files"
	default:
		if inv.Payload.Text != "" {
			args = append(args, inv.Payload.Text)
		}
		draggedType = "text"
	}

	// UseSelectedItemNameAndIcon: a bare run (no drop) operates on the
	// current Finder selection.
	if s.meta.UseSelectedItemNameAndIcon && inv.Payload.IsEmpty() {
		if sel := finderSelection(ctx); len(sel) > 0 {
			args = append(args, sel...)
			draggedType = "files"
		}
	}

	cmd := exec.CommandContext(ctx, s.interpreter, args...)
	if s.meta.RunsSandboxed {
		if wrapped, ok := sandboxedCommand(ctx, s.interpreter, args); ok {
			cmd = wrapped
		} else {
			s.consoleLine("warning: sandbox-exec unavailable; running unsandboxed")
		}
	}
	cmd.Dir = s.bundlePath
	cmd.Env = append(os.Environ(),
		"DZ_ACTION_SCRIPT="+s.scriptPath,
		"DZ_EVENT="+event,
		"DZ_TEMP="+tmp,
		"dragged_type="+draggedType,
		"KEY_MODIFIERS="+strings.Join(inv.Payload.Modifiers, ", "),
	)
	for k, v := range inv.Target.Options {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return actions.Result{}, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return actions.Result{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return actions.Result{}, err
	}
	if err := cmd.Start(); err != nil {
		return actions.Result{}, fmt.Errorf("starting %s: %w", s.meta.Name, err)
	}
	// Drain stderr concurrently: a script writing enough stderr to fill the
	// pipe would otherwise deadlock against our stdout-only scan.
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			s.consoleLine("stderr: " + sc.Text())
		}
	}()

	res, scriptErr := s.consumeProtocol(ctx, stdout, stdin, inv)
	stdin.Close()
	waitErr := cmd.Wait()
	<-stderrDone
	if scriptErr != nil {
		s.runFailed()
		return actions.Result{}, scriptErr
	}
	if waitErr != nil {
		s.runFailed()
		return actions.Result{}, fmt.Errorf("%s exited abnormally: %w", s.meta.Name, waitErr)
	}
	return res, nil
}

// consoleLine forwards one line of raw script output to the debug console.
func (s *ScriptAction) consoleLine(line string) {
	if s.host.Console != nil {
		s.host.Console(line)
	}
}

// runFailed reports a failed script run (for console auto-open).
func (s *ScriptAction) runFailed() {
	if s.host.RunFailed != nil {
		s.host.RunFailed()
	}
}

// consumeProtocol parses DZX: lines from the script's stdout, applying
// progress and side effects as they stream in. stdin is used to answer
// interactive requests (inputbox).
func (s *ScriptAction) consumeProtocol(ctx context.Context, r io.Reader, stdin io.Writer, inv actions.Invocation) (actions.Result, error) {
	var res actions.Result
	var failMsg string
	var dropBarPaths []string

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		rest, ok := strings.CutPrefix(line, "DZX:")
		if !ok {
			// Plain puts/print output goes to the debug console, like
			// Dropzone's; it carries no protocol meaning.
			s.consoleLine(line)
			continue
		}
		kind, payload, _ := strings.Cut(rest, ":")
		payload = strings.ReplaceAll(payload, "", "\n")
		a, b, _ := strings.Cut(payload, "")
		switch kind {
		case "BEGIN", "FINISH":
			inv.Progress.Detail(payload)
			if kind == "FINISH" {
				res.Message = payload
			}
		case "DETERMINATE":
			if payload == "false" {
				inv.Progress.Percent(-1)
			}
		case "PERCENT":
			if p, err := strconv.Atoi(payload); err == nil {
				inv.Progress.Percent(p)
			}
		case "URL":
			if payload != "" {
				res.URL = payload
				_ = inv.Services.CopyToClipboard(payload)
			}
		case "TEXT":
			if payload != "" {
				_ = inv.Services.CopyToClipboard(payload)
			}
		case "FAIL", "ERROR":
			if kind == "ERROR" {
				failMsg = strings.TrimSpace(a + ": " + b)
			} else {
				failMsg = payload
			}
		case "ALERT":
			inv.Services.Notify(a, b)
		case "SAVE":
			if s.host.SaveValue != nil {
				s.host.SaveValue(inv.Target.ID, a, b)
			}
		case "REMOVE":
			if s.host.RemoveValue != nil {
				s.host.RemoveValue(inv.Target.ID, payload)
			}
		case "DROPBAR":
			dropBarPaths = append(dropBarPaths, payload)
		case "INPUTBOX":
			answer, ok := "", false
			if s.host.RequestInput != nil {
				answer, ok = s.host.RequestInput(ctx, a, b)
			}
			if !ok {
				answer = ""
			}
			fmt.Fprintln(stdin, strings.ReplaceAll(answer, "\n", ""))
		}
	}
	if len(dropBarPaths) > 0 && s.host.AddDropBar != nil {
		s.host.AddDropBar(dropBarPaths)
	}
	if failMsg != "" {
		return actions.Result{}, fmt.Errorf("%s", failMsg)
	}
	if err := sc.Err(); err != nil {
		return actions.Result{}, err
	}
	return res, nil
}

// LoadDir scans dir for *.dzbundle directories and returns script actions.
func LoadDir(dir string, host Host) ([]*ScriptAction, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*ScriptAction
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".dzbundle") {
			continue
		}
		action, err := LoadBundle(filepath.Join(dir, e.Name()), host)
		if err != nil {
			// A broken bundle must not break startup, but say why it was skipped.
			log.Printf("bundles: skipping %s: %v", e.Name(), err)
			continue
		}
		out = append(out, action)
	}
	return out, nil
}

// LoadBundle loads one .dzbundle directory.
func LoadBundle(bundlePath string, host Host) (*ScriptAction, error) {
	interpreter, scriptPath := "", ""
	if p := filepath.Join(bundlePath, "action.rb"); exists(p) {
		interpreter, scriptPath = "ruby", p
	} else if p := filepath.Join(bundlePath, "action.py"); exists(p) {
		interpreter, scriptPath = "python3", p
	} else {
		return nil, fmt.Errorf("%s: no action.rb or action.py", bundlePath)
	}
	meta, err := ParseMeta(scriptPath)
	if err != nil {
		return nil, err
	}
	if meta.UniqueID == "" {
		meta.UniqueID = filepath.Base(bundlePath)
	}
	if meta.MinDropzoneVersion != "" && VersionNewer(meta.MinDropzoneVersion, CurrentAppVersion) {
		return nil, fmt.Errorf("%s requires DragZone %s or newer (running %s)",
			meta.Name, meta.MinDropzoneVersion, CurrentAppVersion)
	}
	// Tag console output with the action name so interleaved runs stay
	// readable in the shared debug console.
	if host.Console != nil {
		base := host.Console
		host.Console = func(line string) { base(meta.Name + ": " + line) }
	}
	var iconB64 string
	if data, err := os.ReadFile(filepath.Join(bundlePath, "icon.png")); err == nil {
		iconB64 = base64.StdEncoding.EncodeToString(data)
	}
	return &ScriptAction{
		bundlePath:  bundlePath,
		scriptPath:  scriptPath,
		interpreter: interpreter,
		meta:        meta,
		iconB64:     iconB64,
		host:        host,
	}, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
