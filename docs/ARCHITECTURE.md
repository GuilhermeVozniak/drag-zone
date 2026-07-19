# DragZone Architecture

DragZone is a Wails v2 application: a Go backend owns all state and behavior,
a React frontend renders the grid, and a cgo/Objective-C bridge provides the
macOS integrations Wails does not. This document is the map; per-package
detail lives in the package doc comments.

## Module map

```
cmd/dz ───── unix socket ───┐
                              ▼
frontend ◀─ bindings/events ─ App facade (package main)
                              │  app.go          construction & native wiring
                              │  app_grid.go     targets, drops, tasks
                              │  app_dropbar.go  the shelf
                              │  app_bundles.go  .dzbundle hosting
                              │  app_ipc.go      dz CLI dispatch
                              │  app_settings.go settings, dialogs, window
                              ▼
 ┌──────────┬───────────┬────────────┬────────────┐
 internal/  internal/     internal/      internal/
 {config,   actions       tasks          platform (cgo)
  grid,     ├─ builtin/   (Runner:       ├─ bridge_darwin.{h,m,go}
  dropbar}  └─ registry    async exec,   └─ services_darwin.go
  + storage internal/      progress,
  (JSON)    bundles        notifications)
            (.dzbundle host)
```

Dependency rules: `main` imports `internal/*`; `actions/builtin` may import
`actions`, `model`, `fsutil`, `platform`; `bundles` imports `actions`/`model`;
nothing imports `main`. `platform` is the only package containing cgo.

## Core concepts

- **ActionSpec** (`internal/model`) — an installable action type: id, name,
  icon, events (`dragged`/`clicked`), accepted payload kinds, option fields.
- **Target** — an action instance placed in the grid, with per-instance
  options (folder path, credentials, …), a position, and an optional
  single-key shortcut.
- **Payload** — what lands on a target: file paths, text, or a URL, plus
  modifier keys held during the drop (`Option` inverts folder copy/move).
- **Action** (`internal/actions`) — the engine interface. `Dropper` handles
  drops, `Clicker` handles clicks. Actions receive an `Invocation`:
  target + payload + `Progress` (detail/percent) + `Services` (host
  capabilities: clipboard, notify, open, reveal, trash, AirDrop) +
  `SaveOption` (persist rotated credentials on the target).
- **Runner** (`internal/tasks`) — executes actions in goroutines, tracks
  `TaskState`s, streams them to the frontend, fires notifications. Built from
  a `tasks.Config` (emit, services, notify predicate, option saver).

## Data flows

**Native file drop** → Wails `EnableFileDrop` delivers paths + coordinates →
frontend `useNativeFileDrop` resolves the element under the cursor by
`data-drop-id` → `DropOnTarget(id, payload)` or Drop Bar stash → registry
lookup → `Runner.Run` → `tasks:changed` events → progress rows in the grid.

**Drag-out** → mousedown+move on a Drop Bar file tile → `StartDragOut(item)`
→ `platform.StartDrag` begins an `NSDraggingSession` from the current mouse
event → session end reports back → `DropBarConsume` removes the item unless
locked or the keep-items setting is on.

**Menu bar** → the status item overlay view handles click (toggle grid),
right-click (menu), and `NSDraggingDestination` drops (stash + open grid).
A global monitor watches for file drags near the menu bar and shows the grid
passively. The grid hides when the app deactivates, unless pinned (popped-out
Drop Bar).

## Protocols

**Frontend events** (Go → JS, names defined in `app.go`, subscribed in
`frontend/src/lib/backend.ts`):

| Event | Payload | Meaning |
|---|---|---|
| `grid:changed` | `Target[]` | targets added/edited/moved/removed |
| `dropbar:changed` | `Item[]` | shelf contents changed |
| `tasks:changed` | `TaskState[]` | task started/progressed/finished |
| `specs:changed` | `ActionSpec[]` | bundle installed/developed |
| `settings:open` | tab | enter settings mode on the given tab |
| `settings:close` | — | settings mode closed |
| `window:visibility` | `bool` | native show/hide of the grid |
| `dropbar:popout` | `bool` | pop-out mode toggled |
| `input:request` | `{id,title,prompt}` | script inputbox needs an answer (reply via `AnswerInputRequest`) |

**Script protocol** (`internal/bundles`): the Ruby/Python shims translate the
`$dz`/`dz` API into `DZX:<KIND>:<payload>` lines on stdout (`\x1e` escapes
newlines, `\x1f` separates fields). `INPUTBOX` blocks the script on stdin
until the app writes the user's answer. Items arrive as argv; options and
`dragged_type`/`KEY_MODIFIERS` as environment variables.

**CLI IPC** (`internal/ipc`): one JSON `{cmd, args, flags}` request per
connection on `~/Library/Application Support/DragZone/dragzone.sock`
(chmod 0600), answered with `{ok, data|error}`. `cmd/dz` is the client.

## Persistence

All state is JSON in `~/Library/Application Support/DragZone/` (override with
`$DRAGZONE_DATA_DIR`, used by tests): `settings.json`, `targets.json`,
`dropbar.json`, plus `Actions/` for installed `.dzbundle`s. Writes are atomic
(CreateTemp + rename) and files are created 0600 since target options may
hold credentials.

## Native bridge (internal/platform)

`bridge_darwin.{h,m,go}` exposes: status item + drag detection, activation
policy (Dock show/hide), grid show/hide/position, settings window mode
(re-chromes the shared window into a titled app window and back),
`NSDraggingSession` drag-out, AirDrop via `NSSharingService`, Finder icons
via `NSWorkspace` (base64 PNG), login item via `SMAppService`, Carbon F-key
global hotkey, Option-key state, pinned mode, and ImageIO metadata
stripping. Callbacks re-enter Go through `//export`ed functions and are
dispatched on new goroutines.
`services_darwin.go` implements `actions.Services` with system tools
(pbcopy/pbpaste, osascript, open, lp, qlmanage via the facade).

## Known tradeoffs

- SFTP uses `ssh.InsecureIgnoreHostKey` (personal-tool tradeoff).
- Credentials live in `targets.json` (0600); Keychain is future work.
- The pop-out Drop Bar reuses the single Wails window (v2 has no multi-window)
  by switching the UI into a compact pinned mode. Settings also reuses it:
  opening settings flips the window into a titled, centered app window
  (Dock icon shown) and back on close — the grid is unavailable meanwhile.
- `pashua` script dialogs are unsupported; `inputbox` is.
