package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"dragzone/internal/addons"
	"dragzone/internal/bundles"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
	"dragzone/internal/platform"
	"dragzone/internal/storage"
)

// Scriptable .dzbundle action hosting: installation, authoring, and the
// interactive capabilities running scripts can call back into.

// inputTimeout bounds how long a script's inputbox waits for the user.
const inputTimeout = 5 * time.Minute

type inputAnswer struct {
	Value string
	OK    bool
}

// actionsDir is where .dzbundle actions are installed.
func actionsDir() (string, error) {
	base, err := storage.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Actions")
	return dir, os.MkdirAll(dir, 0o755)
}

// bundleHost exposes app capabilities to running bundle scripts.
func (a *App) bundleHost() bundles.Host {
	return bundles.Host{
		SaveValue: a.saveTargetOption,
		RemoveValue: func(targetID, name string) {
			a.saveTargetOption(targetID, name, "")
		},
		AddDropBar: func(paths []string) {
			_, _ = a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: paths})
		},
		RequestInput: a.requestInput,
		Console:      a.appendConsole,
		RunFailed:    func() { a.emit(EventConsoleError) },
	}
}

// appendConsole adds one line to the debug console buffer and notifies the
// frontend. The buffer is capped; oldest lines drop off.
func (a *App) appendConsole(line string) {
	a.consoleMu.Lock()
	a.consoleLines = append(a.consoleLines, ConsoleLine{Line: line, At: time.Now()})
	if len(a.consoleLines) > consoleCap {
		a.consoleLines = a.consoleLines[len(a.consoleLines)-consoleCap:]
	}
	lines := make([]ConsoleLine, len(a.consoleLines))
	copy(lines, a.consoleLines)
	a.consoleMu.Unlock()
	a.emit(EventConsoleChanged, lines)
}

// ConsoleLines returns the debug console buffer.
func (a *App) ConsoleLines() []ConsoleLine {
	a.consoleMu.Lock()
	defer a.consoleMu.Unlock()
	lines := make([]ConsoleLine, len(a.consoleLines))
	copy(lines, a.consoleLines)
	return lines
}

// ClearConsole empties the debug console.
func (a *App) ClearConsole() {
	a.consoleMu.Lock()
	a.consoleLines = nil
	a.consoleMu.Unlock()
	a.emit(EventConsoleChanged, []ConsoleLine{})
}

// inputRequest is the payload sent to the frontend for a dz.inputbox text
// prompt or a choice prompt (e.g. file-conflict resolution). When Choices is
// set the frontend renders buttons instead of a text field.
type inputRequest struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Prompt  string   `json:"prompt"`
	Choices []string `json:"choices,omitempty"`
}

// requestInput shows a text-input dialog and blocks the caller (a running
// action script) until the user answers, the task is cancelled, or the
// timeout passes.
func (a *App) requestInput(ctx context.Context, title, prompt string) (string, bool) {
	return a.promptUser(ctx, title, prompt, nil)
}

// requestChoice shows a button dialog and returns the chosen label; it backs
// Invocation.Prompt so built-in actions can resolve file conflicts.
func (a *App) requestChoice(ctx context.Context, title, message string, choices []string) (string, bool) {
	return a.promptUser(ctx, title, message, choices)
}

func (a *App) promptUser(ctx context.Context, title, prompt string, choices []string) (string, bool) {
	id := uuid.NewString()
	ch := make(chan inputAnswer, 1)
	a.inputMu.Lock()
	a.inputReqs[id] = ch
	a.inputMu.Unlock()
	defer func() {
		a.inputMu.Lock()
		delete(a.inputReqs, id)
		a.inputMu.Unlock()
	}()

	platform.ShowGrid(true)
	a.emit(EventInputRequest, inputRequest{ID: id, Title: title, Prompt: prompt, Choices: choices})

	select {
	case ans := <-ch:
		return ans.Value, ans.OK
	case <-ctx.Done():
		return "", false
	case <-time.After(inputTimeout):
		return "", false
	}
}

// AnswerInputRequest resolves a pending script input dialog.
func (a *App) AnswerInputRequest(id, value string, ok bool) {
	a.inputMu.Lock()
	ch := a.inputReqs[id]
	a.inputMu.Unlock()
	if ch != nil {
		select {
		case ch <- inputAnswer{Value: value, OK: ok}:
		default:
		}
	}
}

// InstallBundle copies a .dzbundle into the actions directory and registers
// it immediately.
func (a *App) InstallBundle(path string) error {
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	dst, err := fsutil.CopyPath(path, dir, nil)
	if err != nil {
		return fmt.Errorf("installing bundle: %w", err)
	}
	act, err := bundles.LoadBundle(dst, a.bundleHost())
	if err != nil {
		os.RemoveAll(dst)
		return err
	}
	if err := a.registry.TryRegister(act); err != nil {
		os.RemoveAll(dst)
		return fmt.Errorf("an action with this ID is already installed")
	}
	a.emit(EventSpecsChanged, a.registry.Specs())
	return nil
}

// DevelopAction creates a template .dzbundle, registers it, and reveals the
// script for editing — the "Develop Action…" workflow.
func (a *App) DevelopAction(name, language string) error {
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	bundle, err := bundles.CreateTemplate(dir, name, language)
	if err != nil {
		return err
	}
	act, err := bundles.LoadBundle(bundle, a.bundleHost())
	if err != nil {
		return err
	}
	if err := a.registry.TryRegister(act); err != nil {
		return err
	}
	a.emit(EventSpecsChanged, a.registry.Specs())
	return a.services.OpenPath(bundle)
}

// CopyScriptForEditing duplicates a script action's bundle into the Actions
// directory with a fresh UniqueID, registers the copy, and opens its script
// for editing — Dropzone's tile "Copy and Edit Script".
func (a *App) CopyScriptForEditing(targetID string) error {
	t, err := a.grid.Get(targetID)
	if err != nil {
		return err
	}
	act, err := a.registry.Get(t.ActionID)
	if err != nil {
		return err
	}
	script, ok := act.(*bundles.ScriptAction)
	if !ok {
		return fmt.Errorf("only script actions can be copied for editing")
	}
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	bundle, scriptPath, err := script.CopyForEditing(dir)
	if err != nil {
		return err
	}
	dup, err := bundles.LoadBundle(bundle, a.bundleHost())
	if err != nil {
		os.RemoveAll(bundle)
		return err
	}
	if err := a.registry.TryRegister(dup); err != nil {
		os.RemoveAll(bundle)
		return err
	}
	a.emit(EventSpecsChanged, a.registry.Specs())
	return a.services.OpenPath(scriptPath)
}

// AddonInfo describes one entry of the official add-on catalogue.
type AddonInfo struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// ListAddons returns the add-on actions available from the official
// aptonic/dropzone4-actions repository, marking already-installed ones.
func (a *App) ListAddons() ([]AddonInfo, error) {
	names, err := addons.List(a.ctx)
	if err != nil {
		return nil, err
	}
	dir, err := actionsDir()
	if err != nil {
		return nil, err
	}
	out := make([]AddonInfo, 0, len(names))
	for _, name := range names {
		_, statErr := os.Stat(filepath.Join(dir, name+".dzbundle"))
		out = append(out, AddonInfo{Name: name, Installed: statErr == nil})
	}
	return out, nil
}

// InstallAddon downloads an add-on from the official repository and installs
// it into the grid's action library.
func (a *App) InstallAddon(name string) error {
	base, err := storage.Dir()
	if err != nil {
		return err
	}
	bundle, cleanup, err := addons.FetchBundle(a.ctx, base, name)
	if err != nil {
		return err
	}
	defer cleanup()
	return a.InstallBundle(bundle)
}

// OpenActionsFolder reveals the bundle installation directory in Finder.
func (a *App) OpenActionsFolder() error {
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	return a.services.OpenPath(dir)
}
