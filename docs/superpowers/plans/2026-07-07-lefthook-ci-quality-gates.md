# Lefthook Local Gates + Enhanced CI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce lint + format + tests locally (lefthook pre-commit/pre-push) and in CI so only clean, tested code reaches `main`.

**Architecture:** Add a Go linter (`golangci-lint`), extend biome to the desktop frontend, wire both plus the test suites into `lefthook` hooks (fast local gate) and `.github/workflows/ci.yml` (full gate). Local hooks skip the `wails build`-dependent Go tests; CI runs the full matrix.

**Tech Stack:** lefthook, golangci-lint v2, biome v2, bun + turbo, GitHub Actions.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-07-lefthook-ci-quality-gates-design.md`.
- Branch: `tooling-quality-gates`. Commit messages end with `Co-Authored-By: WOZCODE <contact@withwoz.com>`.
- **Embed coupling:** desktop `package main` `//go:embed frontend/dist` needs `wails build` first; `./internal/...` and `./cmd/...` do NOT. Frontend vitest needs NO `wails build` (test-alias stubs).
- Do NOT modify vendored `apps/desktop/frontend/src/components/ui/*` except via an automated biome format pass.
- `gofmt -l .` (from `apps/desktop`) stays empty; existing web/shared/Go tests stay green.
- Frontend uses **bun**; Go/wails run from `apps/desktop`.

---

### Task 1: golangci-lint config, clean on the current tree

**Files:**
- Create: `.golangci.yml`
- Possibly modify: trivial genuine findings in `apps/desktop/**/*.go` (only if real; prefer config tuning over churn).

- [ ] **Step 1: Install golangci-lint v2 locally** (for running it during this task):

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint --version   # confirm a v2.x build on PATH ($(go env GOPATH)/bin)
```

- [ ] **Step 2: Create `.golangci.yml`** (v2 schema; default linters + misspell):

```yaml
# golangci-lint v2 config. Run from apps/desktop.
version: "2"
run:
  timeout: 5m
linters:
  # v2 default set (errcheck, govet, ineffassign, staticcheck, unused) plus:
  enable:
    - misspell
  exclusions:
    generated: lax
    presets:
      - std-error-handling
    rules:
      # Test assertions and fixtures intentionally ignore some errors.
      - path: _test\.go
        linters:
          - errcheck
formatters:
  enable:
    - gofmt
```

- [ ] **Step 3: Run on the embed-free packages (must pass with no build):**

Run: `cd apps/desktop && golangci-lint run ./internal/... ./cmd/...`
Expected: no findings. If real findings appear, fix the genuine ones; for noisy/opinionated ones, narrow the config. Re-run until clean.

- [ ] **Step 4: Run on the full tree (needs the embed):**

Run: `cd apps/desktop && wails build && golangci-lint run ./...`
Expected: no findings (this includes `package main`). Fix/tune until clean.

- [ ] **Step 5: Confirm gofmt still clean:**

Run: `cd apps/desktop && gofmt -l .`
Expected: prints nothing.

- [ ] **Step 6: Commit** (config + any genuine fixes):

```sh
git add .golangci.yml apps/desktop
git commit -m "build: add golangci-lint v2 config (clean on current tree)"
```

---

### Task 2: Extend biome to the desktop frontend

**Files:**
- Modify: `apps/desktop/frontend/package.json` (add `lint` script)
- Modify: root `package.json` (`lint` filter includes desktop frontend)
- Modify (automated): `apps/desktop/frontend/src/**` reformatted by biome

- [ ] **Step 1: Add a `lint` script** to `apps/desktop/frontend/package.json` scripts:

```json
"lint": "biome check ."
```

- [ ] **Step 2: One-time format pass** so existing files satisfy biome:

Run: `cd apps/desktop/frontend && bunx biome check --write .`
Expected: files reformatted; command exits 0 (no remaining errors). Inspect the diff — it must be pure formatting (no logic changes).

- [ ] **Step 3: Verify the frontend lint is clean:**

Run: `cd apps/desktop/frontend && bunx biome check .`
Expected: no diagnostics.

- [ ] **Step 4: Extend the root `lint` script** in root `package.json`:

```json
"lint": "turbo run lint --filter=@dragzone/web --filter=@dragzone/shared --filter=@dragzone/desktop-frontend"
```

- [ ] **Step 5: Verify the whole lint passes:**

Run: `cd <repo root> && bun run lint`
Expected: 3 tasks successful (web, shared, desktop-frontend), no biome diagnostics.

- [ ] **Step 6: Confirm frontend still builds/tests** (format pass didn't break anything):

Run: `cd apps/desktop && bun run --filter=@dragzone/desktop-frontend test`
Expected: 84 tests pass.

- [ ] **Step 7: Commit** (format pass separate from wiring for a clean history):

```sh
git add apps/desktop/frontend/src
git commit -m "style: biome-format the desktop frontend"
git add apps/desktop/frontend/package.json package.json
git commit -m "build: bring the desktop frontend under biome lint"
```

---

### Task 3: lefthook local hooks

**Files:**
- Create: `lefthook.yml`
- Modify: root `package.json` (devDep `lefthook` + `prepare` script)

- [ ] **Step 1: Add lefthook** as a root devDependency and a `prepare` install hook:

```sh
cd <repo root> && bun add -D lefthook
```
Then add to root `package.json` scripts: `"prepare": "lefthook install"`.

- [ ] **Step 2: Create `lefthook.yml`:**

```yaml
# Git hooks for local quality gates. Installed by `bun install` (prepare ->
# lefthook install) or `bunx lefthook install`.
# pre-push requires golangci-lint on PATH: `brew install golangci-lint`.

pre-commit:
  parallel: true
  commands:
    gofmt:
      glob: "*.go"
      run: gofmt -w {staged_files}
      stage_fixed: true
    biome:
      glob: "*.{js,ts,jsx,tsx}"
      run: bunx biome check --write --no-errors-on-unmatched {staged_files}
      stage_fixed: true

pre-push:
  parallel: false
  commands:
    lint:
      run: bun run lint
    golangci-lint:
      root: "apps/desktop/"
      run: >
        command -v golangci-lint >/dev/null 2>&1 ||
        { echo "pre-push: golangci-lint not found — install with: brew install golangci-lint"; exit 1; }
        && golangci-lint run ./internal/... ./cmd/...
    go-test:
      root: "apps/desktop/"
      run: go test ./internal/... ./cmd/...
    frontend-test:
      run: bun run --filter=@dragzone/desktop-frontend test
    web-shared-test:
      run: bun run test
```

- [ ] **Step 3: Install and smoke-test the hooks:**

Run: `cd <repo root> && bunx lefthook install && bunx lefthook run pre-commit && bunx lefthook run pre-push`
Expected: both hook groups pass on the clean tree (all commands green).

- [ ] **Step 4: Commit:**

```sh
git add lefthook.yml package.json bun.lock
git commit -m "build: add lefthook pre-commit + pre-push quality gates"
```

---

### Task 4: Enhance CI

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Web job** — after `bun run test`, add the desktop-frontend vitest step (biome coverage is already picked up because `bun run lint` now includes it):

```yaml
      - run: bun run --filter=@dragzone/desktop-frontend test
```
(Place it after the existing `- run: bun run test` line, before `bun run build`.)

- [ ] **Step 2: Desktop job** — add golangci-lint AFTER the existing `Build app` (wails build) step and BEFORE `Go tests`:

```yaml
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v2.1.6
          working-directory: apps/desktop
          args: --timeout=5m
```
(Pin to the same v2 line the config targets; adjust the patch version to a real release if v2.1.6 is unavailable.)

- [ ] **Step 3: Sanity-check the YAML** parses:

Run: `cd <repo root> && bunx --yes yaml-lint .github/workflows/ci.yml 2>/dev/null || python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml')); print('yaml ok')"`
Expected: `yaml ok` (or the linter's success).

- [ ] **Step 4: Commit:**

```sh
git add .github/workflows/ci.yml
git commit -m "ci: golangci-lint on desktop + biome/vitest for desktop frontend"
```

---

### Task 5: Full-gate verification + prerequisite note

**Files:**
- Modify: `CLAUDE.md` (or `README`) — short note on the new gates + golangci-lint prereq.

- [ ] **Step 1: Run the local pre-push gate end-to-end:**

Run: `cd <repo root> && bunx lefthook run pre-push`
Expected: all commands pass (lint, golangci-lint, go test internal/cmd, frontend vitest, web/shared tests).

- [ ] **Step 2: Run the full backend gate (CI parity):**

Run: `cd apps/desktop && wails build && golangci-lint run ./... && go test ./...`
Expected: golangci-lint clean; all Go packages pass.

- [ ] **Step 3: Add a Contributor-guide note** in `CLAUDE.md` under Commands: hooks auto-install via `bun install`; `golangci-lint` must be installed locally (`brew install golangci-lint`); what pre-commit/pre-push run; that CI enforces the full matrix.

- [ ] **Step 4: Commit:**

```sh
git add CLAUDE.md
git commit -m "docs: document lefthook gates + golangci-lint prerequisite"
```

---

## Done-when

- `bunx lefthook run pre-commit` and `bunx lefthook run pre-push` pass on a clean tree; hooks auto-install on `bun install`.
- `golangci-lint run ./...` (post `wails build`) and `golangci-lint run ./internal/... ./cmd/...` are clean.
- `bun run lint` covers web + shared + desktop frontend with no biome diagnostics.
- `ci.yml`: web job runs desktop-frontend vitest + biome; desktop job runs golangci-lint + `go test ./...`. YAML valid.
- All existing tests (Go 15 pkgs, frontend 84, web/shared 4) remain green.
