// Command dz controls a running DragZone app, mirroring Dropzone 4's CLI:
//
//	dz list                          list grid targets
//	dz run NAME dragged|clicked [FILES...]
//	dz list-items [--json]           list Drop Bar items
//	dz add [--stack] FILES...        add files to the Drop Bar
//	dz rename INDEX NEW_NAME|--reset rename a Drop Bar item (1-based index)
//	dz remove INDEX | dz lock INDEX | dz unlock INDEX | dz clear
//	dz open | dz close               show/hide the grid
//	dz open-dropbar | dz close-dropbar  pop the Drop Bar out / dock it
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dragzone/internal/ipc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	flags := map[string]bool{}
	var rest []string
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			flags[strings.TrimPrefix(a, "--")] = true
		} else {
			rest = append(rest, a)
		}
	}

	// File arguments become absolute so the app resolves them correctly.
	if cmd == "add" || cmd == "run" {
		for i, a := range rest {
			if abs, err := filepath.Abs(a); err == nil {
				if _, statErr := os.Stat(abs); statErr == nil {
					rest[i] = abs
				}
			}
		}
	}

	data, err := ipc.Call(ipc.Request{Cmd: cmd, Args: rest, Flags: flags})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dz:", err)
		os.Exit(1)
	}
	print_(cmd, data, flags["json"])
}

func print_(cmd string, data json.RawMessage, asJSON bool) {
	if asJSON || cmd == "list-items" && asJSON {
		fmt.Println(string(data))
		return
	}
	switch cmd {
	case "list":
		var targets []struct {
			Label  string `json:"label"`
			Action string `json:"action"`
			Events string `json:"events"`
		}
		if json.Unmarshal(data, &targets) == nil {
			for _, t := range targets {
				fmt.Printf("%-24s %-16s %s\n", t.Label, t.Action, t.Events)
			}
			return
		}
	case "list-items":
		var items []struct {
			Label  string `json:"label"`
			Kind   string `json:"kind"`
			Locked bool   `json:"locked"`
		}
		if json.Unmarshal(data, &items) == nil {
			for i, it := range items {
				lock := ""
				if it.Locked {
					lock = " [locked]"
				}
				fmt.Printf("%d. %s (%s)%s\n", i+1, it.Label, it.Kind, lock)
			}
			return
		}
	}
	var s string
	if json.Unmarshal(data, &s) == nil && s != "" {
		fmt.Println(s)
		return
	}
	if string(data) != "null" {
		fmt.Println(string(data))
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: dz COMMAND [ARGS]

  list                              list grid targets
  run NAME dragged|clicked [FILES]  run an action by its grid label
  list-items [--json]               list Drop Bar items
  add [--stack] FILES               add files to the Drop Bar
  rename INDEX NEW_NAME|--reset     rename a Drop Bar item
  remove INDEX                      remove a Drop Bar item
  lock INDEX | unlock INDEX         toggle keeping an item after drag-out
  clear                             clear the Drop Bar
  open | close                      show or hide the grid
  open-dropbar | close-dropbar      pop out or dock the Drop Bar
`)
}
