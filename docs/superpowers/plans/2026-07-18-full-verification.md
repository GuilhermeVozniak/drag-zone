# DragZone Full Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give DragZone a durable, joint automated regression suite plus a live end-to-end verification of every feature, so future ships cannot silently break existing behavior.

**Architecture:** Six streams. First a live exploration pass (Claude drives the real app via `wails dev` + a browser, producing a discovery report and bug list). Then raised Go coverage (unit + App-facade flow tests using the existing `recServices`/package-var seams), frontend component tests, a new Playwright browser-E2E layer with a mock backend, and finally CI wiring + docs. Native-only interactions stay a documented manual checklist.

**Tech Stack:** Go 1.26 + `go test` (httptest, temp-dir fakes), Wails v2, React 19 + Vitest + Testing Library (jsdom), Playwright + chromium, bun workspaces + turbo, GitHub Actions.

## Global Constraints

- **No live third-party network** in tests — `httptest.Server` / in-process servers only.
- **No real `platform.*` cgo in tests** — fake the `actions.Services` interface; cgo tested only at error/guard branches or via existing `//go:build darwin` tests on the macOS runner.
- Every test touching a store sets `t.Setenv(storage.EnvDataDir, t.TempDir())`. For unix-socket (IPC) tests use `os.MkdirTemp("", "dz")` to stay under the macOS 104-byte `AF_UNIX` path limit.
- Do **not** modify vendored `apps/desktop/frontend/src/components/ui/*` except via the automated `biome check --write` pass.
- All `go`/`wails` commands run from `apps/desktop` (not repo root). Frontend uses **bun**.
- `go test ./...` (package `main`) requires `wails build` first (it regenerates the gitignored `frontend/dist` that `//go:embed` needs). `./internal/...` and `./cmd/...` do not.
- Commit messages end with:
  `Co-Authored-By: WOZCODE <contact@withwoz.com>` and a `Claude-Session:` trailer.
- Work happens on branch `full-verification` (already created).
- Reuse existing test fakes — `recServices` (`builtin_test_helpers_test.go`), `nullProgress`/`dropInv` (`simple_actions_test.go`), `noopServices`/`newTestApp` (`app_grid_test.go`) — do not re-declare them.

---

## Stream 5 — Live exploration (FIRST)

### Task 1: Live self-exploration + discovery report

**This task is run by the main thread (Claude), not a delegated subagent** — it needs the Chrome automation tools and a running native app. It produces a report and, for any bug found, a new bug-fix task appended to this plan.

**Files:**
- Create: `docs/EXPLORATION-2026-07-18.md`
- Create: screenshots under `docs/exploration-assets/` (referenced from the report)

- [ ] **Step 1: Build the CLI and start the dev app**

```sh
cd apps/desktop
go build -o build/bin/dz ./cmd/dz
wails dev            # serves the frontend devserver on http://localhost:34115
```
Run `wails dev` in the background; wait for `http://localhost:34115` to answer.

- [ ] **Step 2: Drive the UI in a real browser**

Load the Chrome tools with one ToolSearch (`tabs_context_mcp,navigate,computer,read_page,tabs_create_mcp,get_page_text`). Open `http://localhost:34115`. Screenshot each screen while walking:
grid empty state; Add-to-Grid + Add-to-Drop-Bar tiles; add a target of each category via the catalogue (folder, app, and each built-in action) and open its config panel; simulate text/URL drops and a `data-drop-id` file drop; watch a TASK PROGRESS row start/progress/finish; Recently Shared pill; Drop Bar stash → stack → rename → lock → clear → pop-out; every Settings tab (General grid-size slider + theme + login item, Add-ons, Command Line, Updates); onboarding carousel.

- [ ] **Step 3: Exercise the CLI against the live app**

```sh
./build/bin/dz list
./build/bin/dz add /etc/hosts
./build/bin/dz list-items --json
./build/bin/dz clear
```
Confirm the grid / Drop Bar in the browser react to each command.

- [ ] **Step 4: Write the report**

Write `docs/EXPLORATION-2026-07-18.md`: a per-feature table (Feature | Expected | Observed | PASS/FAIL | screenshot), a **Bugs found** section, and a **Manual checklist** for native-only interactions (drag-onto-menubar overlay, drag-OUT to Finder, AirDrop sheet, F-key hotkey, tray click) with each marked ✅ verified / ⚠️ needs-user.

- [ ] **Step 5: Append bug-fix tasks**

For each FAIL, append a `### Task 1.x` TDD bug-fix task to this plan (failing test that reproduces the bug → fix → green). Stop the `wails dev` process.

- [ ] **Step 6: Commit**

```sh
git add docs/EXPLORATION-2026-07-18.md docs/exploration-assets
git commit -m "docs: live exploration report + discovery findings"
```

---

## Stream 2 — App-facade flow tests

### Task 2: Make `App.emit` injectable (test seam)

**Files:**
- Modify: `apps/desktop/app.go` (the `emit` method + `App` struct)
- Test: `apps/desktop/app_emit_test.go`

**Interfaces:**
- Produces: `App.onEmit func(event string, data ...any)` — when non-nil, `emit` calls it instead of `runtime.EventsEmit`. Flow tests set it to a recorder.

- [ ] **Step 1: Write the failing test**

```go
package main

import "testing"

func TestEmitUsesOnEmitHook(t *testing.T) {
	app := newTestApp(t)
	var got []string
	app.onEmit = func(event string, _ ...any) { got = append(got, event) }
	app.emit("x:changed", 1)
	if len(got) != 1 || got[0] != "x:changed" {
		t.Fatalf("onEmit not invoked, got %v", got)
	}
}
```

- [ ] **Step 2: Run it — expect FAIL** (`app.onEmit undefined`)

Run: `cd apps/desktop && go test . -run TestEmitUsesOnEmit` (needs `wails build` first on a clean tree).

- [ ] **Step 3: Add the field + branch**

In `App` struct add `onEmit func(event string, data ...any)`. Change `emit`:
```go
func (a *App) emit(event string, data ...any) {
	if a.onEmit != nil {
		a.onEmit(event, data...)
		return
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event, data...)
	}
}
```

- [ ] **Step 4: Run — expect PASS**, and `go test .` still green.

- [ ] **Step 5: Commit** — `test(app): injectable emit hook for flow tests`

### Task 3: Flow test — file drop runs the action end-to-end

**Files:**
- Test: `apps/desktop/app_flow_test.go`

**Interfaces:**
- Consumes: `newTestApp`, `App.onEmit` (Task 2). Use a recording `actions.Services` — declare a local `flowServices` (App-package tests cannot see the `builtin` package's `recServices`).

- [ ] **Step 1: Write the failing test** — add a `savetext`/`clipboard` target, drop text, assert the recording service saw the side effect and `tasks:changed` was emitted.

```go
func TestDropOnTargetRunsActionAndEmits(t *testing.T) {
	app := newTestApp(t)
	var events []string
	done := make(chan struct{})
	app.onEmit = func(ev string, _ ...any) {
		events = append(events, ev)
		if ev == EventTasksChanged { // emitted on completion
			select { case <-done: default: close(done) }
		}
	}
	tgt, err := app.AddTarget("clipboard", "Copy", nil)
	if err != nil { t.Fatal(err) }
	if _, err := app.DropOnTarget(tgt.ID, model.Payload{Kind: model.ItemText, Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("no tasks:changed emitted")
	}
}
```
(Implementer: confirm the exact completion-event constant name in `app.go`; add `EventTasksChanged` to the constants block if it is spelled differently, and read a real no-config action id — `clipboard`, `trash` — from `builtin.go`.)

- [ ] **Step 2: Run — expect FAIL**, then make it pass by fixing any missing constant/id reference (no product logic change expected).
- [ ] **Step 3: Run — expect PASS.**
- [ ] **Step 4: Commit** — `test(app): drop→run→tasks:changed flow`

### Task 4: Flow test — text and URL drops dispatch correctly

**Files:** Test: `apps/desktop/app_flow_test.go` (extend)

- [ ] **Step 1:** Table-driven test: for a `shorten`-style and a `savetext`-style target, assert `DropOnTarget` routes `ItemText` vs `ItemURL` payloads to the action and rejects unaccepted kinds with an error. Use a stub target whose action is a no-config builtin.
- [ ] **Step 2–4:** run FAIL→implement→PASS→commit `test(app): text/url drop routing`.

### Task 5: Flow test — Drop Bar stash → drag-out → consume

**Files:** Test: `apps/desktop/app_dropbar_flow_test.go`

**Interfaces:** Consumes `App.DropBarAdd`, `App.StartDragOut`, `App.DropBarConsume`, `App.SetDropBarKeepItems`/settings (read real method names from `app_dropbar.go`).

- [ ] **Step 1:** Stash two paths via `DropBarAdd`; assert `dropbar:changed` emitted and item count. Begin `StartDragOut(itemID)`; simulate session-end; assert `DropBarConsume` removes the item **unless** locked or keep-items is on (two sub-cases).
- [ ] **Step 2–4:** FAIL→green→commit `test(app): dropbar stash/dragout/consume flow`.

### Task 6: Flow test — `dz` IPC round-trips end-to-end

**Files:** Test: `apps/desktop/app_ipc_flow_test.go`

**Interfaces:** Start the app's IPC server on a temp socket (`os.MkdirTemp("","dz")`); connect a raw client and send `{cmd,args,flags}` JSON. Read the real request/response shape from `internal/ipc` and `app_ipc.go`.

- [ ] **Step 1:** For `list`, `add`, `list-items`, `clear`: send the command, assert `{ok:true}` and the resulting store mutation + emitted event. Assert an unknown command returns `{ok:false,error:...}`.
- [ ] **Step 2–4:** FAIL→green→commit `test(app): dz IPC round-trip flows`.

---

## Stream 1 — Raise Go coverage

General pattern (already established in `imgur_test.go`): point the action's package-var endpoint at an `httptest.Server`, build an `actions.Invocation` with `&recServices{}` + `nullProgress{}` + a temp-file payload, assert result URL / clipboard / `%w`-wrapped errors. Verify each task raises the package's coverage:
`go test ./internal/actions/builtin/ -cover` (capture to a file; the stdout hook collapses it).

### Task 7: Shorten action coverage

**Files:** Modify: `apps/desktop/internal/actions/builtin/shorten_test.go`; read `shorten.go` for the endpoint var.

- [ ] **Step 1:** httptest server returns a short URL; assert clipboard + result URL, non-2xx → error, empty input → error, hotkey/selection path if present.
- [ ] **Step 2–4:** FAIL→green→commit `test(shorten): httptest round-trip + errors`.

### Task 8: S3 action coverage

**Files:** Modify: `s3_test.go`; read `s3.go`. If the endpoint is not already a package var, add one (`var s3EndpointOverride string`) used only when set, defaulting to the real region host.

- [ ] **Step 1:** httptest server accepts a PUT; assert object key, `Option-drop` zips first (payload has `"Option"` modifier → single `.zip` uploaded), missing-credential error, result URL → clipboard.
- [ ] **Step 2–4:** FAIL→green→commit `test(s3): upload + zip-first + credential errors`.

### Task 9: Google Drive coverage (endpoint seam + refresh/upload)

**Files:** Modify: `gdrive.go` (make `driveTokenURL`, `driveUploadURL` package `var`s like `driveFilesURL`), `gdrive_test.go`.

- [ ] **Step 1:** With `refresh_token` preset (skips the browser flow), point token+upload+files URLs at one httptest server that issues a token, accepts the multipart upload, and returns a `webViewLink`. Assert: link copied to clipboard, `SaveOption` called if the refresh token rotates, upload error → `%w` wrap.
- [ ] **Step 2–4:** FAIL→green→commit `test(gdrive): refresh+upload against httptest`.

### Task 10: FTP action coverage (no live server)

**Files:** Modify: `ftp_test.go`; read `ftp.go`.

- [ ] **Step 1:** Cover the non-network logic: missing host/user/pass → error; `url_prefix` result construction + clipboard; protocol dispatch (`ftp` vs `sftp`) selects the right dialer and a dial failure is `%w`-wrapped. Do **not** stand up a live FTP/SFTP server (network constraint) — assert via a seam or the fast error paths.
- [ ] **Step 2–4:** FAIL→green→commit `test(ftp): config/url_prefix/dispatch coverage`.

### Task 11: Folder action — conflict resolution + Option-invert

**Files:** Modify: `folder_test.go`; read `folder.go`, `fsutil`.

- [ ] **Step 1:** Into a temp dst that already contains the file, script `Invocation.Prompt` to return each of `Keep Both` / `Replace` / `Stop` and assert the resulting filesystem state; `Prompt==nil` → safe default (Keep Both); `Option` modifier inverts copy↔move; progress reported by bytes.
- [ ] **Step 2–4:** FAIL→green→commit `test(folder): conflict dialog + option-invert`.

### Task 12: Filesystem actions — zip / savetext / convert / metadata

**Files:** Modify: `zip_test.go`, `savetext_test.go`; Create: `convert_test.go`, `metadata_test.go`.

- [ ] **Step 1:** zip → assert a real archive is produced from temp files and unzips to the same bytes; savetext → asks name (via `Prompt`/option) and writes the file; convert/metadata **guard cgo** — test only the input-validation and error branches reachable without the darwin bridge (e.g. unsupported-format error, empty payload), and skip the cgo path with `t.Skip` when `runtime.GOOS != "darwin"` only if a real conversion is attempted.
- [ ] **Step 2–4:** FAIL→green→commit `test(builtin): zip/savetext/convert/metadata`.

### Task 13: Services-backed actions — openapp / airdrop / print / clipboard / trash

**Files:** Modify: `apps_test.go`, `simple_actions_test.go`; Create tests for any uncovered one.

- [ ] **Step 1:** With `&recServices{}`, assert each action calls the right service method with the right args: openapp click → `OpenPath`/launch, openapp drop → open-with; airdrop → `AirDrop(paths)`; print → the print service; clipboard → `CopyToClipboard`; trash → `Trash(paths)`. Assert Spec events/accepts.
- [ ] **Step 2–4:** FAIL→green→commit `test(builtin): services-backed actions`.

### Task 14: Install Application coverage (exec seam)

**Files:** Modify: `installapp.go` (route `hdiutil`/`cp` through a package-var command runner `var runCmd = exec.CommandContext`), `installapp_test.go` (new).

- [ ] **Step 1:** Inject a fake `runCmd` that records invocations and returns canned output; assert the mount→copy→launch→eject→trash sequence and that a mount failure aborts with a `%w` error. No real `.dmg`.
- [ ] **Step 2–4:** FAIL→green→commit `test(installapp): mount/copy/eject sequence via exec seam`.

### Task 15: Add-ons catalogue coverage

**Files:** Modify: `internal/addons/addons_test.go`; read `addons.go` for the catalogue URL seam (add a package `var` if absent).

- [ ] **Step 1:** httptest server serves a catalogue JSON + a fake bundle zip; assert parse, install-to-`Actions/` (temp data dir), and a malformed-catalogue error.
- [ ] **Step 2–4:** FAIL→green→commit `test(addons): catalogue parse + install`.

### Task 16: CLI (`cmd/dz`) coverage

**Files:** Modify: `cmd/dz/main_test.go`.

- [ ] **Step 1:** Table-driven: run each subcommand (`list`, `run`, `add`, `list-items`, `rename`, `remove`, `lock`, `unlock`, `clear`, `open`, `close`) against a fake IPC server on a temp socket; assert the JSON request the CLI sends and its rendering of the response; assert a connection error is reported cleanly.
- [ ] **Step 2–4:** FAIL→green→commit `test(dz): subcommand request/response coverage`.

### Task 17: fsutil + storage error/conflict paths

**Files:** Modify: `internal/fsutil/fsutil_test.go`, `internal/storage/storage_test.go`.

- [ ] **Step 1:** fsutil: `CopyPathAs`/`MovePathAs` exact-dst, unique-name generation on conflict, dir recursion, error on unreadable source. storage: atomic write (temp+rename) leaves no partial file on a simulated failure, 0600 perms, load-missing returns the zero value, corrupt-JSON error.
- [ ] **Step 2–4:** FAIL→green→commit `test(fsutil,storage): conflict + atomic-write paths`.

---

## Stream 3 — Frontend component tests (vitest)

Pattern: render with `@testing-library/react`, stub the backend via the existing `src/test/stubs/{App,runtime}.ts` aliases; assert render + that user actions call the right backend method (spy) and event subscriptions update the UI. Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test`.

### Task 18: Drop Bar components

**Files:** Create: `frontend/src/features/dropbar/{DropBarTile,TopSection,PopoutBar}.test.tsx`.

- [ ] **Step 1:** DropBarTile renders name/thumb, double-click → Quick Look call, mousedown+move → `StartDragOut`; TopSection renders stash tiles + scroll affordance at overflow; PopoutBar renders compact pinned mode and accepts drops.
- [ ] **Step 2–4:** FAIL→green→commit `test(frontend): drop bar components`.

### Task 19: Grid components

**Files:** Create: `frontend/src/features/grid/{GridPanel,TargetTile,RecentSharesPill}.test.tsx`.

- [ ] **Step 1:** GridPanel click routing (run|config|none per `clickBehavior`), Option-delete jiggle class + remove call; TargetTile renders icon/label/shortcut and the key-modifier glyph while a drag hovers; RecentSharesPill renders newest-first and opens a URL on click.
- [ ] **Step 2–4:** FAIL→green→commit `test(frontend): grid components`.

### Task 20: Settings components

**Files:** Create: `frontend/src/features/settings/{AddonsTab,CommandLineTab,DevelopActionRow,SettingsDialog}.test.tsx`.

- [ ] **Step 1:** AddonsTab lists the catalogue + install button calls backend; CommandLineTab install-CLI button; DevelopActionRow generates a template; SettingsDialog switches tabs and persists changes via the backend.
- [ ] **Step 2–4:** FAIL→green→commit `test(frontend): settings components`.

---

## Stream 4 — Browser E2E (Playwright)

### Task 21: Playwright scaffold + mock backend

**Files:**
- Create: `frontend/e2e/mock/backend.ts` — implements the `window.go.main.App` methods + `window.runtime` (EventsEmit/On) used by `src/lib/backend.ts`, backed by in-memory grid/dropbar/tasks/settings state.
- Create: `frontend/e2e/fixtures.ts` — injects the mock via `page.addInitScript` before app load.
- Create: `frontend/playwright.config.ts` — `webServer` runs `vite build --mode e2e && bunx serve dist -l 4173` (or `vite preview`), `baseURL: http://localhost:4173`.
- Modify: `frontend/vite.config.ts` — under an `e2e` mode, alias the `wailsjs` specifiers to `./e2e/mock/backend.ts` (same regex-alias mechanism the test block already uses), so `vite build` succeeds on ubuntu without generated bindings.
- Modify: `frontend/package.json` — add `"e2e": "playwright test"` and `@playwright/test` devDep.

**Interfaces:**
- Produces: `installMockBackend(page)` fixture; the mock exposes the same method names `src/lib/backend.ts` imports (read that file for the exact surface).

- [ ] **Step 1:** Read `src/lib/backend.ts`; enumerate every `App.*` method and `runtime` call it uses. Implement them in `backend.ts` with in-memory state + event dispatch.
- [ ] **Step 2:** Add the `e2e` vite mode alias; confirm `bunx vite build --mode e2e` produces a `dist/` that boots with the mock (no wailsjs).
- [ ] **Step 3:** Add `playwright.config.ts` + a trivial smoke spec (`e2e/smoke.spec.ts`: grid root renders). Run `cd apps/desktop/frontend && bunx playwright install --with-deps chromium && bun run e2e` — expect PASS.
- [ ] **Step 4:** Commit — `test(e2e): playwright scaffold + mock backend`.

### Task 22: E2E — grid add / configure / drop / progress

**Files:** Create: `frontend/e2e/grid.spec.ts`.

- [ ] **Step 1:** Open grid → click Add-to-Grid → catalogue → pick an action → configure options → save; assert the tile appears. Trigger a drop through the mock (dispatch the drop path `useNativeFileDrop` listens for) → assert a TASK PROGRESS row appears, advances, finishes, and the Recently Shared pill updates.
- [ ] **Step 2:** Run `bun run e2e` — expect PASS. Commit `test(e2e): grid add/config/drop/progress`.

### Task 23: E2E — drop bar / settings / onboarding

**Files:** Create: `frontend/e2e/dropbar-settings.spec.ts`.

- [ ] **Step 1:** Drop Bar stash → merge into a stack → rename → clear; Settings: switch each tab, move grid-size slider, toggle theme, assert the mock persisted; onboarding carousel next/prev/dismiss (gated on `OnboardingSeen`).
- [ ] **Step 2:** Run `bun run e2e` — expect PASS. Commit `test(e2e): dropbar + settings + onboarding`.

---

## Stream 6 — CI wiring + docs

### Task 24: Wire desktop E2E into CI

**Files:** Modify: `.github/workflows/ci.yml` (the `web` job).

- [ ] **Step 1:** After the desktop-frontend vitest step, add:
```yaml
      - name: Desktop frontend E2E (playwright)
        run: |
          cd apps/desktop/frontend
          bunx playwright install --with-deps chromium
          bun run e2e
```
- [ ] **Step 2:** Validate the YAML (`ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml")'`). Do **not** add E2E to the lefthook pre-push (keep local fast; CI covers it).
- [ ] **Step 3:** Commit — `ci: run desktop frontend playwright e2e in the web job`.

### Task 25: Docs + final green verification

**Files:** Create: `docs/TESTING.md`; Modify: `CLAUDE.md` (## Commands, ## Quality gates).

- [ ] **Step 1:** `docs/TESTING.md` — the layer table (unit / flow / component / E2E / manual checklist) and the exact command to run each locally.
- [ ] **Step 2:** Update `CLAUDE.md` Commands with the new `e2e` scripts and note the manual checklist lives in the exploration report.
- [ ] **Step 3: Full verification** — from `apps/desktop`: `wails build` then `go test ./...`; `bun run --filter=@dragzone/desktop-frontend test`; `cd frontend && bun run e2e`; from repo root `bun run test` and `bun run lint`. Capture each result to a file (the stdout hook collapses `go test`). All must be green.
- [ ] **Step 4:** Commit — `docs: testing guide + commands; verify full suite green`.

---

## Self-Review

**Spec coverage:** Stream 1 → Tasks 7–17; Stream 2 → Tasks 2–6; Stream 3 → Tasks 18–20; Stream 4 → Tasks 21–23; Stream 5 → Task 1; Stream 6 → Tasks 24–25. Every spec success criterion maps to a task (builtin ≥75% → Tasks 7–14; App flow tests → 3–6; Playwright green → 21–23; docs → 25; discovery report → 1).

**Placeholder scan:** Coverage tasks name exact files, seams, and scenarios; scaffolding tasks (2, 14, 21, 24) carry literal code/YAML. Where an exact symbol must be read from real code (event constant name in Task 3, backend surface in Task 21), the step says so explicitly rather than inventing a name — this is a deliberate read-then-implement instruction, not a placeholder.

**Type consistency:** `App.onEmit` (Task 2) is consumed by Tasks 3–6. `recServices`/`nullProgress`/`newTestApp` are reused, not redeclared (Global Constraints). Package-var seams (`imgurAPIURL` exists; `driveTokenURL`/`driveUploadURL`, `s3EndpointOverride`, `runCmd`, addons catalogue URL) are introduced in the task that needs them.
