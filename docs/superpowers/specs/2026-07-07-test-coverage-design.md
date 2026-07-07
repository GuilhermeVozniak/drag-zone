# DragZone — Complete Test Coverage Design

**Date:** 2026-07-07
**Status:** Approved (design), pending implementation plan

## Goal

Bring the DragZone monorepo to comprehensive test coverage across every
functionality area, with the coverage gap concentrated in the desktop app
(Go backend + React frontend). `apps/web` and `packages/shared` already have
tests; the desktop app has only scattered coverage today.

Scope decision (confirmed): **Full — logic + components.** Cover all Go
packages and the App facade, all built-in actions (network ones via injectable
endpoints + httptest), tasks/ipc/config/model/storage, plus frontend `lib/`,
`hooks/`, and the logic-bearing feature components (adding the missing React
test infra).

## Current coverage baseline

Tested today:
- Go: `app_grid`, `app_settings`, `actions/builtin/{folder,zip}`, `bundles`
  (partial), `dropbar/store`, `fsutil`, `grid/store`, `platform/encode_darwin`.
- Frontend: `features/grid/clickBehavior` only.
- Web + shared: `download`, `landing` e2e, `shared/index`.

Untested (the gap this design closes):
- Go pure-logic: `config`, `model`, `storage`, `ipc`, `addons`,
  `tasks/runner`, `bundles/{action,meta,shims,template}`, `actions/actions.go`.
- Go App facade: `app.go`, `app_bundles`, `app_dropbar`, `app_ipc`, `cmd/dz`.
- Built-in actions: `airdrop`, `clipboard`, `convert`, `ftp`, `gdrive`,
  `imgur`, `installapp`, `metadata`, `openapp`, `print`, `s3`, `savetext`,
  `shorten`, `trash`.
- Frontend: all of `lib/` (except clickBehavior), all `hooks/`, all
  `features/` components.

## 1. Test infrastructure additions

**Frontend** (`apps/desktop/frontend`):
- Add devDeps: `jsdom`, `@testing-library/react`, `@testing-library/user-event`,
  `@testing-library/jest-dom`.
- Add `vitest.config.ts` with `test.environment: "jsdom"`, `globals: true`,
  and a setup file registering `@testing-library/jest-dom` matchers and
  auto-cleanup. Preserve the existing `@` path alias from `vite.config.ts`.
- Add a manual mock for the generated Wails bindings. `lib/backend.ts` is the
  only importer of `../../wailsjs/*`, so a single mock module for
  `wailsjs/go/main/App`, `wailsjs/runtime/runtime`, and `wailsjs/go/models`
  lets every hook/component test run without the Wails runtime. Tests drive
  behavior by configuring the mock's return values and firing mocked events.

**Go**: no new infra. Reuse the existing `noopServices` fake, `httptest`, and
`t.Setenv(storage.EnvDataDir, t.TempDir())` patterns. Promote `noopServices`
(and any channel-synced recording fake) into a shared test helper if it is
needed across more than one `package main` test file.

## 2. Go backend coverage

**Pure logic** — direct unit tests:
- `model`: `Payload.HasModifier`, `Payload.IsEmpty`, `Target.Option`
  (incl. empty-value fallthrough).
- `config`: load/defaults/save round-trip, settings merge.
- `storage`: `Save`/`Load` round-trip under a temp `EnvDataDir`, missing-file
  and malformed-JSON handling.
- `ipc`: request/response encode/decode over the control channel.
- `addons`: listing/install logic.
- `bundles/{action,meta,shims,template}`: manifest parse, template expansion,
  shim wiring (extend existing `bundles_test.go`).
- `actions/actions.go`: registry `Register`/`TryRegister` duplicate handling,
  `Get` unknown-id error, `Specs` ordering and non-nil guarantee.

**App facade** — via `newTestApp` + `noopServices`:
- `app.go`: recent-shares cap at 10 + newest-first + persistence,
  `taskFeedback` running-count transitions, `saveTargetOption` set/delete,
  `ClearRecentShares`, `RecentShares` copy independence.
- `app_dropbar.go`, `app_bundles.go`, `app_ipc.go`: the bound methods'
  mutations and emitted events.
- `cmd/dz`: CLI argument handling against a fake IPC server.

**tasks/runner**: dragged/clicked dispatch selection, unsupported-event error,
error path (notify + "Basso" sound), success path (notify + "Glass" gated on
settings), `Cancel`, `Dismiss` (only finished tasks), progress `Detail`/
`Percent` propagation, `OnResultURL` callback. Use a fake `Action` and a
channel-synchronized fake `Services` so tests are deterministic without real
sleeps.

**Built-in actions**: pure helpers directly (`isImageFile`, `parseHTTPURL`,
folder copy/move mode, zip contents, savetext, metadata, convert arg-building).
Network actions per section 3.

## 3. Network actions — injectable endpoints

Decision (confirmed): **small refactor for injectable endpoints.**

For `imgur`, `shorten`, `s3`, `ftp`, `gdrive`, replace each hardcoded endpoint
with an unexported package-level `var` (e.g. `imgurEndpoint = "https://..."`)
or an endpoint field, defaulting to the production URL. Tests point it at an
`httptest.NewServer` and assert:
- request building (method, path, query),
- auth headers (e.g. `Authorization: Client-ID ...`),
- multipart/body encoding,
- success response parsing (extracted link/URL),
- error paths (non-200 status, empty/malformed body).

This mirrors the existing `app_settings_test.go` update-check pattern
(`checkForUpdates(ts.URL)`). Production behavior is unchanged; the var simply
becomes overridable from `_test.go`. FTP/S3 that cannot hit a plain httptest
server are covered at the request-construction/credential-validation boundary
plus any pure helpers; the network I/O seam is made injectable where feasible.

## 4. Frontend coverage

**Pure logic / lib** (no DOM): `lib/dnd.ts`, `lib/icons.ts`, `lib/utils.ts`,
`backend.ts` `uiScale` (clamp + scale formula). `clickBehavior` already done.

**Hooks** (`renderHook` + mocked backend/events):
- `useBackend`: subscribes on mount, updates state on emitted
  grid/tasks/dropbar/specs/shares events, unsubscribes on unmount.
- `useFileIcon`: caching + backend call.
- `useNativeFileDrop`: native drop wiring.
- `useTargetShortcuts`: single-key dispatch to the matching target, ignores
  when grid closed / input focused.

**Feature components** (render + `user-event`, backend mocked) — the
interactive, logic-bearing ones:
`AddTargetDialog`, `OptionsForm`, `TargetTile`, `GridPanel`, `DropBarTile`,
`RenameItemDialog`, `InputRequestDialog`, `TaskList`, `GeneralTab`,
`UpdatesTab`, `AddonsTab`, `CommandLineTab`, `Onboarding`. Assert the visible
behavior and the backend calls each triggers, not internal structure.

## 5. Out of scope

1. Native cgo/Objective-C calls in `platform/bridge_darwin.{m,go}` and
   `services_darwin.go` — cannot run headless. Only Go-side pure helpers are
   covered (e.g. `encode_darwin`, already tested; add tests for any other pure
   helper found there).
2. shadcn `components/ui/*` primitives — vendored, kept untouched per CLAUDE.md.
3. Purely-presentational wrappers with no logic (e.g. `PanelChrome`,
   `SettingRow`, `ActionIcon`) unless they carry conditional logic worth
   pinning.
4. `apps/web` / `packages/shared` — already covered; fill only obvious gaps.
5. Live third-party network I/O — never hit real Imgur/TinyURL/S3/Drive; all
   round-trips go through httptest or injectable seams.

## Verification

All must be green (✅ = confirmed on the final green-gate run, 2026-07-07):
- [x] `apps/desktop`: `wails build` (regenerates `frontend/dist` for the embed),
  then `go test ./...` — **15 packages pass**.
- [x] `bun run --filter=@dragzone/desktop-frontend test` (frontend vitest) —
  **18 files / 84 tests pass**.
- [x] `bun run test` (web + shared vitest via turbo) — **4 tests pass**.
- [x] `gofmt -l .` prints nothing.
- [x] `cd apps/desktop/frontend && bunx tsc --noEmit` clean (test files excluded).
- [x] `bun run lint` (biome) clean.

## Outcome

Delivered via `superpowers:subagent-driven-development` (one implementer +
task-review per task on branch `test-coverage`):

- **Go backend** — direct unit/integration tests for `model`, `storage`,
  `config`, `ipc`, `actions/registry`, every builtin action
  (clipboard/trash/airdrop/save-text/install-app/ftp + the five network
  actions imgur/shorten/addons/s3/gdrive via injectable endpoints + httptest),
  `bundles`, `tasks/runner`, the `App` facade (shares/dropbar/ipc), and the
  `dz` CLI.
- **Frontend** — new Vitest + jsdom + Testing Library harness with a manual
  `@/lib/backend` mock; coverage of `lib/` (`cn`/`uiScale`/`dnd`/`icons`),
  every hook, and the logic-bearing feature components.
- **Bug found by the new tests:** `App.RecentShares()` returned `nil` (JSON
  `null`) instead of `[]Share{}` when empty — fixed to guard on `len() == 0`.
- **Small production seams** (behavior-preserving): injectable endpoint `var`s
  for imgur/shorten/addons/gdrive, and an extracted `s3PublicURL` helper.
- Native cgo, shadcn `components/ui/*`, and presentational wrappers remain out
  of scope as planned.
