# DragZone

A feature-parity clone of [Dropzone 4](https://aptonic.com) for macOS, built with
[Wails v2](https://wails.io) (Go) and React + [shadcn/ui](https://ui.shadcn.com).

DragZone lives in your menu bar. Drag files onto the menu bar icon (or press the
global shortcut, F3 by default) and a grid of actions slides down: drop files on
a folder to copy/move them, on an app to open them, on AirDrop, Zip, Imgur, S3,
FTP/SFTP, and more. Stash files in the Drop Bar and drag them back out to any
app. Extend it with Dropzone-compatible Ruby/Python `.dzbundle` actions.

## Features

- **Menu bar grid** — slides down under the status item; click the icon or press
  the shortcut; drag a file near the menu bar and it appears automatically.
- **Drop Bar** — temporary shelf holding references to files; multi-file drops
  become stacks; lock items to keep them after dragging out; native drag-out to
  Finder and other apps.
- **Targets** — folders (copy/move, Option inverts), applications (drop to open
  with, click to launch), and actions, all rearrangeable and configurable with
  per-target options; real Finder icons.
- **Built-in actions** — Move/Copy to Folder, Open Application, AirDrop,
  Zip Files, Copy to Clipboard, Move to Trash, Install Application (.dmg/.zip),
  Save Text, Print, Shorten URL (TinyURL), Imgur Upload, Amazon S3 Upload,
  FTP/SFTP Upload.
- **Task engine** — concurrent tasks with progress bars in the grid,
  notifications on completion, result URLs copied to the clipboard.
- **Scriptable actions** — Dropzone 4-compatible `.dzbundle` format: `action.rb`
  / `action.py` with the `# Dropzone Action Info` header, `$dz` / `dz` API
  (begin, percent, finish, url, text, fail, error, alert, save_value,
  add_dropbar, …), OptionsNIB config panels, `KEY_MODIFIERS`, installed in
  `~/Library/Application Support/DragZone/Actions`.
- **Settings** — launch at login (SMAppService), global F-key shortcut, grid
  columns, notification toggles.

## Building

Requirements: macOS 13+, Go 1.23+, Node 20+, the Wails v2 CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).

```sh
wails build                        # produces build/bin/dragzone.app
wails dev                          # live-reload development
go build -o build/bin/dz ./cmd/dz  # the dz command line tool
go test ./internal/...             # backend tests
```

## Documentation

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — module map, data flows, protocols
- [`docs/ACTIONS.md`](docs/ACTIONS.md) — writing `.dzbundle` actions and the `dz` CLI
- [`docs/DROPZONE4-PARITY.md`](docs/DROPZONE4-PARITY.md) — the Dropzone 4 feature spec this clone tracks
- [`CLAUDE.md`](CLAUDE.md) — contributor guide (commands, conventions, gotchas)

## Architecture

```
main.go                     wails bootstrap, window options
app*.go                     bindings facade, split by domain (grid, dropbar,
                            bundles, ipc, settings)
internal/
  model/      shared domain types (Payload, Target, ActionSpec, TaskState)
  actions/    action engine: interfaces, registry, host services
  actions/builtin/  built-in action implementations
  bundles/    .dzbundle scriptable action host (Ruby/Python shims)
  tasks/      async task runner streaming progress events
  grid/       target placement + persistence
  dropbar/    shelf state + persistence
  config/     user settings
  storage/    JSON persistence in ~/Library/Application Support/DragZone
  fsutil/     copy/move primitives with byte progress
  platform/   macOS native bridge (cgo/Objective-C): status item, drag-out,
              AirDrop, file icons, login item, global hotkey
frontend/src/
  features/grid|dropbar|tasks|settings   UI features
  components/ui                          shadcn components
  lib/ hooks/                            bindings facade, DnD, live state
```
