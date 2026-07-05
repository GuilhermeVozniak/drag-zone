# DragZone — Contributor Guide

Dropzone 4 clone: Wails v2 (Go) menu bar app + React/TS/shadcn frontend +
cgo/Objective-C macOS bridge. Read `docs/ARCHITECTURE.md` for the module map,
`docs/ACTIONS.md` for the action API, `docs/DROPZONE4-PARITY.md` for the
feature spec being cloned.

## Commands (run from the repo root — always)

```sh
wails build                        # full app → build/bin/dragzone.app
wails dev                          # live-reload development
wails generate module              # regenerate frontend/wailsjs after changing App bindings
go build -o build/bin/dz ./cmd/dz  # the dz CLI companion
go test ./internal/...             # backend tests
cd frontend && npm run build       # frontend typecheck + bundle only
gofmt -l . | grep -v frontend      # must print nothing
```

Gotchas:
- `wails` commands FAIL from `frontend/` with a misleading
  `frontend/wails.json: no such file` error. Run them from the root.
- After adding/renaming any bound `App` method or bound struct field, run
  `wails generate module` before touching the frontend.
- `frontend/wailsjs` and `frontend/dist` are generated and gitignored;
  `wails build` recreates both.
- Tests that touch stores must set `t.Setenv(storage.EnvDataDir, t.TempDir())`
  or they will write to the real `~/Library/Application Support/DragZone`.

## Conventions

- The `App` facade (package main) is split by domain: `app.go` (wiring),
  `app_grid.go`, `app_dropbar.go`, `app_bundles.go`, `app_ipc.go`,
  `app_settings.go`. Put new bindings in the matching file; emit the
  corresponding `Event*` constant after every mutation.
- One built-in action per file in `internal/actions/builtin/`; implement
  `Spec()` plus `Dropped`/`Clicked`, wrap errors with `%w` and context,
  report progress by bytes where possible. Register in `builtin.go`.
- Actions must not reach into app state; use `Invocation.Services`,
  `Invocation.SaveOption`, and options via `Target.Option`.
- All cgo lives in `internal/platform`. New native capabilities: declare in
  `bridge_darwin.h`, implement in `bridge_darwin.m` (main-thread via
  dispatch), wrap in `bridge_darwin.go`; Go callbacks via `//export` must
  spawn goroutines.
- Frontend: `lib/backend.ts` is the only file that imports wailsjs; events
  are subscribed there and consumed through hooks in `hooks/`. Feature UI
  lives under `features/<area>/`, one component per file; shadcn primitives
  in `components/ui` stay untouched.
- Frontend event names must match the `Event*` constants in `app.go`.
