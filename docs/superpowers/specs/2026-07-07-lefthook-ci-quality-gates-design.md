# Lefthook Local Gates + Enhanced CI — Design Spec

## Goal

Guarantee that only well-formed, linted, tested code lands on `main`, enforced
in two places:

1. **Locally** via `lefthook` git hooks (fast feedback before commit/push).
2. **In CI** (`.github/workflows/ci.yml`) as the authoritative gate.

Close three gaps in the current setup: no Go linter (only `gofmt -l` + `go
test`), the desktop frontend (`apps/desktop/frontend`) is unlinted, and its 84
vitest tests never run in CI.

## Context / constraints

- Bun monorepo + turbo. `apps/web` (Next.js), `apps/desktop` (Wails v2 Go +
  React frontend), `packages/shared`. Biome config at root; `bun run lint`/`test`
  currently filter to web + shared only.
- **Embed coupling:** the desktop `package main` `//go:embed frontend/dist`
  requires `wails build` (which also regenerates `frontend/wailsjs`) before
  `go test .` compiles. `internal/*` and `cmd/dz` do NOT need the embed or
  `wailsjs`.
- **Frontend vitest** uses the Task 21 test-alias stubs for `wailsjs/*`, so it
  runs on a clean checkout with NO `wails build` (ubuntu-friendly).
- **Frontend tsc typecheck** DOES need generated `wailsjs` types — already
  covered by `wails build` (`bun run build` = `tsc && vite build`); no separate
  step required.

## Decisions (confirmed)

- **Pre-push depth:** *fast local + full CI*. The local hook skips the
  `wails build`-dependent App-facade Go tests and full-tree lint; CI runs the
  full matrix.
- **Biome scope:** *extend to the desktop frontend* via a one-time
  `biome check --write` format pass (its own commit).
- **Hook stages:** *pre-commit* (fast format/lint on staged files) **and**
  *pre-push* (the lint + test gate).
- **Go linter:** `golangci-lint` v2, curated linter set.

## Components

### 1. `lefthook.yml` (new, repo root)

**pre-commit** — staged files, autofix + re-stage (`stage_fixed: true`):
- `gofmt -w` on staged `*.go`
- `biome check --write` on staged `*.{js,ts,jsx,tsx}`

**pre-push** — the gate (no `wails build`; ~15–25s):
- `biome check` over web + shared + desktop frontend
- `golangci-lint run ./internal/... ./cmd/...` (run from `apps/desktop`)
- `go test ./internal/... ./cmd/...` (run from `apps/desktop`)
- `bun run --filter=@dragzone/desktop-frontend test` (vitest)
- `bun run test` (web + shared)

Commands that shell out to `golangci-lint` fail with a clear “install
golangci-lint” message when the binary is absent.

### 2. `.golangci.yml` (new)

golangci-lint **v2** config. Curated enabled linters: `govet`, `staticcheck`,
`errcheck`, `ineffassign`, `unused`, `misspell` (plus any that pass cleanly).
The config must pass on the current tree — tune the set / add narrow excludes
rather than mass-editing existing code, but fix trivial genuine findings.

### 3. Root `package.json`

- devDependency: `lefthook`.
- `"prepare": "lefthook install"` — hooks install on `bun install`.
- Extend `lint` to include the desktop frontend:
  `turbo run lint --filter=@dragzone/web --filter=@dragzone/shared --filter=@dragzone/desktop-frontend`.

### 4. `apps/desktop/frontend/package.json`

- Add `"lint": "biome check ."` (resolves the root `biome.json` by walking up).
- turbo `lint` `dependsOn: ["^build"]`; the desktop frontend has no workspace
  build deps, so biome runs directly (no `wails`/`tsc` needed for lint).

### 5. One-time format pass

`biome check --write apps/desktop/frontend/src` (+ config/test files as biome
selects) — committed separately as the initial reformatting so subsequent
`biome check` passes.

### 6. `.github/workflows/ci.yml`

**`web` job (ubuntu):**
- `bun run lint` now also covers the desktop frontend (via the extended filter).
- Add `bun run --filter=@dragzone/desktop-frontend test` (vitest; no wailsjs
  needed).
- Existing web/shared test, build, e2e steps unchanged.

**`desktop` job (macos):**
- After the existing `wails build`, add `golangci-lint run ./...` (via
  `golangci/golangci-lint-action`, pinned v2) — `main` now compiles because
  `dist`/`wailsjs` exist.
- Existing gofmt + `go test ./...` steps unchanged.

## Out of scope

- Rearchitecting the existing turbo web/shared lint/test wiring.
- Adding new lint RULES to biome (`preset: none` stays — biome is format +
  syntax enforcement here).
- Pre-push running the App-facade / embed-dependent Go tests (CI covers them).
- Auto-installing `golangci-lint` locally (documented prerequisite).

## Verification

- `bunx lefthook install` succeeds; `lefthook run pre-commit` and
  `lefthook run pre-push` both pass on the clean tree.
- `golangci-lint run ./...` (from `apps/desktop`, after `wails build`) is clean;
  `golangci-lint run ./internal/... ./cmd/...` is clean without a build.
- `biome check` passes across web + shared + desktop frontend after the format
  pass.
- `apps/desktop/frontend` vitest (84 tests) + web/shared tests still green.
- CI YAML is valid (both jobs) and mirrors the local gate plus the full matrix.
