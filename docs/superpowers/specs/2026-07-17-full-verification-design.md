# DragZone — Full Verification: Design Spec

**Date:** 2026-07-17
**Status:** Approved (design phase)
**Author:** Guilherme Vozniak + Claude

## Problem

DragZone is feature-complete for Dropzone-4 parity, and a large unit-test
epic has landed (all suites green). But coverage is uneven and the app's most
user-facing behaviors are the least verified:

| Area | Coverage | Why low |
|---|---|---|
| `internal/platform` (cgo bridge) | 10% | Native drag/drop, menu bar, AirDrop — real macOS + manual interaction |
| `internal/actions/builtin` | 35% | Real action logic — network / cgo / system tools |
| `internal/addons` | 29% | Catalogue install flow |
| `cmd/dz` (CLI) | 49% | |
| `internal/fsutil` / `storage` | 59% / 64% | Error/conflict paths |
| App facade (`package main`) | partial | Needs `wails build` first |
| Most other `internal/*` | 82–100% | Well covered |

Additionally, **no one has driven the assembled app end-to-end** to confirm
features and UI actually work together. The goal is two things:

- **A. A durable, joint regression suite** we run on every future ship, so we
  cannot silently break existing features.
- **B. A live exploration/discovery pass** where Claude runs the real app in a
  sandbox, verifies every feature and the UI, fixes what's broken, and a new
  **end-to-end test layer** codifies those flows.

## Goals

1. Raise automated coverage across everything reasonably testable, without
   live network I/O or real cgo.
2. Add browser E2E (Playwright) driving the real rendered desktop UI.
3. Add Go flow/integration tests exercising complete drop→runner→event flows
   through the App facade.
4. Perform a live self-exploration of the running app; produce a discovery
   report + screenshots; fix bugs found (with regression tests).
5. Wire everything into CI and the local pre-push gate; keep both green.

## Non-goals (YAGNI / scope guard)

- Automating the raw Objective-C bridge or real OS-level drag/drop/menu-bar
  clicks in CI. These are not automatable; they stay a **documented manual
  checklist** executed as far as the sandbox allows and flagged for the user.
- Rewriting existing passing tests or vendored `components/ui/*` (only touched
  by the automated biome format pass).
- New product features beyond fixing bugs surfaced during exploration.

## Standing constraints (must hold in every task)

- **No live third-party network** — use `httptest` / in-process servers only.
- **No real `platform.*` cgo in tests** — fake the `Services` interface; cgo
  code is tested only at its error/guard branches or on the darwin CI runner
  via existing `//go:build darwin` tests.
- Tests touching stores set `t.Setenv(storage.EnvDataDir, t.TempDir())` (use
  the short-path `os.MkdirTemp("", "dz")` helper for unix-socket tests to stay
  under the macOS 104-byte AF_UNIX limit).
- Do not modify vendored `apps/desktop/frontend/src/components/ui/*` except via
  the automated biome format pass.
- Commit messages end with the `Co-Authored-By: WOZCODE` + `Claude-Session`
  trailers; PR bodies end with the WOZCODE line; branch first off `main`.

## Architecture of the work — six streams

Execution order: **Stream 5 first** (discovery grounds everything), then
Streams 1–4 (tests + fixes), then Stream 6 (CI). Streams 1–4 are largely
independent and can interleave.

### Stream 5 — Live self-exploration (first)

**Sandbox:** `wails dev` from `apps/desktop` runs the real Go backend and
serves the frontend on the Vite devserver (`http://localhost:34115`). That URL
renders the same React app wired to the live backend over the dev websocket,
so it is drivable in a normal browser via the Chrome automation tools and
screenshottable — independent of the native window's show/hide state.

**Procedure:** walk every feature and screenshot each screen —
- Grid: open, empty state, persistent Add-to-Grid / Add-to-Drop-Bar tiles.
- Add a target of each category via the catalogue (folder, app, each built-in
  action); configure options; confirm the config panel / SkipConfig behavior.
- Trigger drops the devserver can simulate (text/URL drops, and file drops via
  the `data-drop-id` hit-test path where drivable); watch TASK PROGRESS rows
  appear, progress, and finish; Recently Shared pill updates.
- Drop Bar: stash, merge into a stack, rename, lock, clear, pop-out mode.
- Settings: every tab (General grid-size slider + theme + login item, Add-on
  catalogue, Command Line installer, Updates check); onboarding carousel.
- `dz` CLI: build `dz`, run `list` / `run` / `add` / `list-items` / `clear`
  against the live app socket; confirm grid/dropbar react.
- Cross-check colors/layout/states against `docs/UX-SPEC.md`.

**Native-only interactions** (real drag-onto-menubar overlay, drag-**out** to
Finder, AirDrop share sheet, F-key global hotkey, tray-icon click) cannot be
driven from the browser. They go into a **manual verification checklist** in
the report, executed where the sandbox permits and otherwise flagged.

**Output:** `docs/EXPLORATION-2026-07-17.md` — per-feature PASS/FAIL, bugs
found, screenshots, and the manual checklist. Every bug found is fixed under
the relevant stream with a regression test.

### Stream 1 — Raise Go coverage (unit/integration)

Target `internal/actions/builtin` 35% → ~75% and lift the other low packages:

- **Network actions** (imgur, shorten, s3, gdrive, ftp): drive each against an
  `httptest.Server` (or in-process FTP/SFTP server) injected via the action's
  configurable endpoint/client seam. Assert request shape, success → clipboard
  URL, and error/`%w`-wrapped failure paths.
- **Filesystem actions** (folder, zip, savetext, convert, metadata): run
  against a temp-dir fake filesystem + a fake `Services`. Cover conflict
  resolution (Keep Both / Replace / Stop via a scripted `Prompt`), Option-drop
  inversion, and zip-before-upload. `convert`/`metadata` wrap cgo — test the
  non-cgo control flow and error branches only.
- **Services-backed actions** (openapp, airdrop, print, clipboard, trash):
  assert they call the right `Services` method with the right args via a
  recording fake; no real cgo.
- `internal/addons`: catalogue parse + install flow against `httptest`.
- `cmd/dz`: more subcommands against a fake IPC server (table-driven).
- `internal/fsutil`, `internal/storage`: conflict, atomic-write-failure, and
  permission error paths.

### Stream 2 — Go flow/integration tests (App facade, package main)

Build a real `App{}` with `NewApp(fakeServices{})` +
`t.Setenv(storage.EnvDataDir, …)` and assert **complete flows**, capturing
emitted events through a fake emitter:

- File / text / URL `DropOnTarget` → registry lookup → `Runner.Run` →
  `tasks:changed` sequence (started → progress → finished) → `grid:changed`.
- Drop Bar: stash → `StartDragOut` → session-end → `DropBarConsume` honoring
  lock / keep-items settings.
- IPC: `dz` commands dispatched through `app_ipc` end-to-end over a real
  socket → state mutation → event emission.
- Settings mutations persist to JSON and emit their events.

These run in the macOS CI `go test ./...` step (after `wails build` provides
the `frontend/dist` embed).

### Stream 3 — Frontend component tests (vitest)

Cover the untested feature components with the existing stub-backend pattern
(`src/test/stubs/{App,runtime}.ts`): `DropBarTile`, `PopoutBar`, `TopSection`,
`GridPanel`, `TargetTile`, `RecentSharesPill`, `AddonsTab`, `CommandLineTab`,
`DevelopActionRow`, `SettingsDialog`. Assert render, event wiring, and the
backend calls each makes.

### Stream 4 — Browser E2E (Playwright, new for desktop frontend)

Mirror the apps/web Playwright setup in `apps/desktop/frontend/`:

- `playwright.config.ts` with a `webServer` that serves a Playwright-only Vite
  build. That build aliases the generated `wailsjs` modules to a **mock
  backend** (`e2e/mock/backend.ts`) implementing the `window.go/main/App` +
  `window.runtime` surface with in-memory grid/dropbar/tasks/settings state
  and event emission — so the *real* React app runs against a deterministic,
  network-free backend in headless chromium.
- Specs in `apps/desktop/frontend/e2e/`: open grid → add target via catalogue
  → configure options → simulate a drop → progress row appears → completes;
  Recently Shared pill; Drop Bar stash / stack / rename / clear; settings tabs
  (grid-size, theme, shortcuts); onboarding carousel next/prev/dismiss.
- Script: `"e2e": "playwright test"` in the desktop-frontend package.

Runs in the **ubuntu web CI job**, which already installs Playwright chromium.

### Stream 6 — CI wiring + docs

- Add a `Desktop frontend E2E (playwright)` step to the `web` job in
  `.github/workflows/ci.yml` (after the vitest step), reusing the existing
  `playwright install --with-deps chromium`.
- Keep the lefthook pre-push gate green (it already runs desktop-frontend
  vitest + web/shared tests; do not add the heavy Playwright/e2e to pre-push —
  CI covers it, per the fast-local / full-CI split).
- No hard coverage-percentage gate initially (avoids flaky failures); we track
  coverage as a reported number and can add a floor later.
- Update `CLAUDE.md` (## Quality gates / ## Commands) and add a short
  `docs/TESTING.md` describing the layers (unit / flow / component / E2E /
  manual checklist) and how to run each.

## Testing strategy summary

| Layer | Tool | Runs in | Covers |
|---|---|---|---|
| Unit | `go test` | both CI jobs / pre-push | package logic, actions, fsutil |
| Flow/integration | `go test` (pkg main) | macOS CI | App facade full flows |
| Component | vitest + jsdom | web CI / pre-push | React components in isolation |
| Browser E2E | Playwright + mock backend | web CI | real UI end-to-end |
| Manual checklist | human | on-demand | native drag/drop, menu bar, AirDrop |
| Live discovery | `wails dev` + browser tools | one-time (this epic) | whole-app verification + UI |

## Risks

- **`wails dev` in this sandbox** may hit macOS permission prompts or need the
  native window; where a native interaction can't be driven, it's documented
  for the user rather than faked. Best-effort, honestly reported.
- **Mock-backend drift** — the Playwright mock must track the real `App`
  binding surface. Mitigation: keep the mock small and typed against the
  generated `wailsjs` d.ts where possible; the Go flow tests cover the real
  backend so the mock only needs behavioral fidelity for UI flows.
- **Playwright build coupling** — the desktop frontend normally needs
  `wails generate module` for real bindings; the E2E build must alias those to
  the mock so it builds on ubuntu without Go/wails. Verified feasible via the
  same alias mechanism the vitest config already uses.

## Success criteria

- Discovery report committed with per-feature results + screenshots; all bugs
  found are fixed with regression tests.
- `internal/actions/builtin` and the other low packages materially higher
  (builtin ~75%+); App-facade flow tests present.
- Playwright desktop E2E green locally and in the ubuntu CI job.
- `bun run test`, `go test ./...`, vitest, and the new E2E all green; pre-push
  and CI green.
- Docs updated (`CLAUDE.md`, `docs/TESTING.md`).
