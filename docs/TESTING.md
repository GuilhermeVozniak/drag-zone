# DragZone — Testing Guide

Six layers cover the app, from fast Go unit tests up to a manual native
checklist for what no automated tool can drive. Run the earlier layers most
often; the later ones are slower or require more setup.

| Layer | Tool | Command | Runs in |
|---|---|---|---|
| Unit (Go) | `go test` | `go test ./internal/... ./cmd/...` (from `apps/desktop`) | any machine, no `wails build` needed |
| App-facade flow/integration (Go, package main) | `go test` | `wails build` then `go test ./...` (from `apps/desktop`) | needs the generated `frontend/dist` embed — build first |
| Component (React) | Vitest | `bun run --filter=@dragzone/desktop-frontend test` (repo root) or `bun run test` (from `apps/desktop/frontend`) | jsdom, stubbed Wails bindings |
| Browser E2E (Playwright + mock backend) | Playwright | `cd apps/desktop/frontend && bun run e2e` | headless chromium against a built app served with a stateful in-memory mock, no Go process involved |
| Web + shared | Vitest | `bun run test` (repo root) | web landing page + `packages/shared` |
| Manual native checklist | human + real app | build `wails build` → `build/bin/dragzone.app`, then walk the checklist | see `docs/EXPLORATION-2026-07-18.md` |

## Notes and constraints

- **No live network in tests.** Anything that would hit a real HTTP endpoint
  (GitHub releases, TinyURL, cloud upload APIs, etc.) is exercised through a
  fake/injected URL or a stubbed transport — never the real internet.
- **No real cgo in tests.** The macOS Objective-C bridge
  (`internal/platform`) is not exercised by `go test`; native-only behavior
  (menu-bar icon, Finder drag/drop, AirDrop, global hotkeys, Quick Look) is
  covered solely by the manual checklist.
- **Frontend E2E uses a mock backend, not the real Go app.** `bun run e2e`
  runs `vite build --mode e2e`, which aliases `wailsjs/go/main/App` and
  `wailsjs/runtime/runtime` to the stateful mock in
  `apps/desktop/frontend/e2e/mock/backend.ts` (see `vite.config.ts`). This
  keeps the suite runnable on Linux CI with no Wails toolchain, but it means
  E2E does not prove the real Go backend's behavior — that's the App-facade
  layer's job.
- **App-facade tests need a build first.** `go test ./...` (package `main`)
  embeds `frontend/dist` via `//go:embed`; on a clean tree that directory
  doesn't exist until `wails build` runs. `go test ./internal/... ./cmd/...`
  has no such dependency and is what `lefthook`'s pre-push hook runs for
  speed.
- **Tests that touch stores** must set
  `t.Setenv(storage.EnvDataDir, t.TempDir())`, or they will read/write the
  real `~/Library/Application Support/DragZone`.

## Quick reference

```sh
# From apps/desktop
go test ./internal/... ./cmd/...          # fast unit tests
wails build && go test ./...              # full suite incl. App-facade tests
gofmt -l .                                 # must print nothing

# From apps/desktop/frontend
bun run test                               # component tests (vitest)
bun run e2e                                 # browser E2E (playwright, mock backend)

# From repo root
bun run test                               # web + shared unit tests
bun run lint                                # biome check web + shared + desktop frontend
```

For what none of the above can reach (menu-bar icon, native Finder
drag/drop, drag-out to Finder, AirDrop, global F-key hotkeys), run the
manual checklist in `docs/EXPLORATION-2026-07-18.md` against a real
`wails build` output.
