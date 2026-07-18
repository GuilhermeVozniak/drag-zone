# DragZone — Contributor Guide

Bun monorepo:
- `apps/desktop` — the Dropzone 4 clone: Wails v2 (Go) menu bar app +
  React/TS/shadcn frontend + cgo/Objective-C macOS bridge.
- `apps/web` — the marketing landing page: Next.js + shadcn, static-exported
  to GitHub Pages (see `.github/workflows/deploy-web.yml`).
- `packages/shared` — the release-asset naming contract used by both the web
  download links and the release workflow.

Read `docs/ARCHITECTURE.md` for the desktop module map (paths there are
relative to `apps/desktop`), `docs/ACTIONS.md` for the action API,
`docs/DROPZONE4-PARITY.md` for the feature spec, `docs/RELEASE.md` for signing
and notarization.

## Commands

Monorepo (from repo root — bun workspaces + turbo):

```sh
bun install                        # install every workspace's deps
bun run build                      # build the landing page (apps/web)
bun run test                       # web + shared unit tests (vitest)
bun run lint                       # biome check web + shared + desktop frontend
```

Desktop app (run from `apps/desktop` — always):

```sh
wails build                        # full app → build/bin/dragzone.app
wails dev                          # live-reload development
wails generate module              # regenerate frontend/wailsjs after changing App bindings
go build -o build/bin/dz ./cmd/dz  # the dz CLI companion
go test ./...                      # backend + App-facade tests
bun run --filter=@dragzone/desktop-frontend test   # frontend unit tests (vitest)
gofmt -l .                         # must print nothing
```

Desktop frontend E2E (run from `apps/desktop/frontend`):

```sh
bun run e2e                        # Playwright against a built app + mock backend (vite --mode e2e)
```

See `docs/TESTING.md` for the full test-layer table (unit / App-facade /
component / E2E / web+shared / manual). The manual native-interaction
checklist (menu-bar icon, Finder drag/drop, AirDrop, hotkeys) lives in
`docs/EXPLORATION-2026-07-18.md`.

Gotchas:
- `wails` and `go` commands run from `apps/desktop`, not the repo root
  (`go.work` lets `go` find the module from the root, but `wails` needs the
  app dir; from `frontend/` it fails with a misleading `wails.json` error).
- The frontend uses **bun** (see `wails.json` `frontend:install`/`build`).
- After adding/renaming any bound `App` method or bound struct field, run
  `wails generate module` before touching the frontend.
- `apps/desktop/frontend/{wailsjs,dist}` are generated and gitignored;
  `wails build` recreates both. `go test` embeds `frontend/dist`, so run
  `wails build` before `go test` on a clean tree.
- The landing page's advertised version is pinned in
  `apps/web/lib/download.ts` (`APP_VERSION`) — bump it in lockstep with a
  release tag so the download link resolves.
- Tests that touch stores must set `t.Setenv(storage.EnvDataDir, t.TempDir())`
  or they will write to the real `~/Library/Application Support/DragZone`.

## Quality gates

`lefthook` runs local git hooks, auto-installed by `bun install` (the root
`prepare` script runs `lefthook install`; run `bunx lefthook install` manually
if hooks are missing):

- **pre-commit** — `gofmt -w` + `biome check --write` on staged files (autofix
  and re-stage).
- **pre-push** — `bun run lint`, `golangci-lint`, fast Go tests
  (`./internal/... ./cmd/...`), the desktop-frontend vitest, and web + shared
  tests. It intentionally skips the `wails build`-dependent App-facade Go tests;
  CI covers those.

pre-push needs **golangci-lint** on PATH: `brew install golangci-lint`.

CI (`.github/workflows/ci.yml`) is the authoritative gate. The web job (ubuntu)
runs lint + web/shared tests + desktop-frontend vitest + build + e2e; the
desktop job (macOS) runs gofmt, `wails build`, `golangci-lint run ./...`, and
`go test ./...`. golangci-lint config lives in `.golangci.yml`.

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
