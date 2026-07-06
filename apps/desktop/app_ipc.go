package main

import (
	"fmt"
	"strconv"
	"strings"

	"dragzone/internal/dropbar"
	"dragzone/internal/ipc"
	"dragzone/internal/model"
	"dragzone/internal/platform"
)

// handleIPC dispatches `dz` CLI commands (see cmd/dz for the user-facing
// syntax). One request maps to one binding-equivalent operation.
func (a *App) handleIPC(req ipc.Request) (any, error) {
	argAt := func(i int) string {
		if i < len(req.Args) {
			return req.Args[i]
		}
		return ""
	}
	switch req.Cmd {
	case "list":
		type row struct {
			Label  string `json:"label"`
			Action string `json:"action"`
			Events string `json:"events"`
		}
		var rows []row
		for _, t := range a.grid.List() {
			events := ""
			if act, err := a.registry.Get(t.ActionID); err == nil {
				events = strings.Join(act.Spec().Events, ", ")
			}
			rows = append(rows, row{Label: t.Label, Action: t.ActionID, Events: events})
		}
		return rows, nil
	case "run":
		return a.ipcRun(argAt(0), argAt(1), req.Args)
	case "list-items":
		return a.dropBar.List(), nil
	case "add":
		if len(req.Args) == 0 {
			return nil, fmt.Errorf("no files given")
		}
		if req.Flags["stack"] {
			if _, err := a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: req.Args}); err != nil {
				return nil, err
			}
			return "added 1 stack", nil
		}
		for _, f := range req.Args {
			if _, err := a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{f}}); err != nil {
				return nil, err
			}
		}
		return fmt.Sprintf("added %d item(s)", len(req.Args)), nil
	case "rename":
		item, err := a.dropBarItemAt(argAt(0))
		if err != nil {
			return nil, err
		}
		name := argAt(1)
		if req.Flags["reset"] {
			name = ""
		}
		return nil, a.DropBarRename(item.ID, name)
	case "remove":
		item, err := a.dropBarItemAt(argAt(0))
		if err != nil {
			return nil, err
		}
		return nil, a.DropBarRemove(item.ID)
	case "lock", "unlock":
		item, err := a.dropBarItemAt(argAt(0))
		if err != nil {
			return nil, err
		}
		return nil, a.DropBarSetLocked(item.ID, req.Cmd == "lock")
	case "clear":
		return nil, a.DropBarClear()
	case "open":
		platform.ShowGrid(true)
		return nil, nil
	case "close":
		platform.HideGrid()
		return nil, nil
	case "open-dropbar":
		return nil, a.SetDropBarPopOut(true)
	case "close-dropbar":
		return nil, a.SetDropBarPopOut(false)
	default:
		return nil, fmt.Errorf("unknown command %q", req.Cmd)
	}
}

// ipcRun executes `dz run NAME EVENT [FILES...]` by grid label.
func (a *App) ipcRun(name, event string, args []string) (any, error) {
	if event != model.EventDragged && event != model.EventClicked {
		return nil, fmt.Errorf("event must be dragged or clicked")
	}
	var target *model.Target
	for _, t := range a.grid.List() {
		if strings.EqualFold(t.Label, name) {
			target = &t
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("no grid target named %q", name)
	}
	payload := model.Payload{}
	if len(args) > 2 {
		payload = model.Payload{Kind: model.ItemFiles, Paths: args[2:]}
	}
	if _, err := a.trigger(target.ID, payload, event); err != nil {
		return nil, err
	}
	return "running " + target.Label, nil
}

// dropBarItemAt resolves a 1-based CLI index into an item.
func (a *App) dropBarItemAt(arg string) (dropbar.Item, error) {
	items := a.dropBar.List()
	if len(items) == 0 {
		return dropbar.Item{}, fmt.Errorf("the Drop Bar is empty")
	}
	idx, err := strconv.Atoi(arg)
	if err != nil || idx < 1 || idx > len(items) {
		return dropbar.Item{}, fmt.Errorf("invalid item index %q (1-%d)", arg, len(items))
	}
	return items[idx-1], nil
}
