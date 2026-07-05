# DragZone — Audit Summary & Refactoring Plan

*2026-07-05, pre-cleanup audit of commit 456ffbb.*

## Executive overview
The codebase is healthy for its size (~8.4k first-party lines): packages are small, single-purpose, and dependency direction is clean (`main` → `internal/*`; `actions/builtin` → `actions`/`model`/`fsutil`/`platform`; nothing imports `main`). The one structural outlier is `app.go` (671 lines), which accumulated seven unrelated domains as the bindings facade grew. The frontend mirrors the backend well (facade + hooks + feature folders); two files outgrew their role.

## Architecture
```
                     ┌─ frontend (React/TS) ────────────────┐
                     │ features/{grid,dropbar,tasks,settings}│
                     │   ↓ hooks/useBackend (events)          │
                     │   ↓ lib/backend.ts (typed facade)      │
                     └────────┬────────────────────────┘
                    wails bindings + runtime events
                              │
cmd/dz ─unix socket─→ internal/ipc ─→ App facade (main pkg)
                              │
        ┌─────────┬─────────┼─────────┬──────────┐
   config/grid/dropbar   actions.Registry   tasks.Runner  platform (cgo)
   (storage: JSON)        ↑          ↑         │            │ status item, drag-out,
                    actions/builtin  bundles ←─┘            │ AirDrop, icons, hotkey,
                    (16 actions)     (.dzbundle host)       │ login item, ImageIO
```
Key flows: drop → frontend resolves tile → `DropOnTarget` → registry → runner (goroutine) → progress events → grid UI. Scripts speak the `DZX:` stdout line protocol; `dz` speaks JSON over the unix socket.

## Code quality assessment
- **Idioms:** good error wrapping (%w with context), mutex-guarded stores, contexts threaded through actions. gofmt/vet clean.
- **Smells found:** god-file facade (app.go); one package-level mutable hook (`builtin.SaveTargetOption`) breaking DI; positional `NewRunner` params; swallowed `Close` error in zip writer; silent skip of broken bundles; `print_` naming in cmd/dz; IPC socket left at default permissions.
- **Pattern check (smell → pattern):** the IPC switch is a flat command dispatcher — fine at this size, no Strategy needed; Registry already implements the registry pattern; actions implement Strategy via the Dropper/Clicker interfaces. No pattern-level restructuring warranted.
- **Coverage:** stores (grid/dropbar/config) untested → MEDIUM risk for store refactors; engine and fs primitives tested.

## Security review
- **MEDIUM:** SFTP uses `ssh.InsecureIgnoreHostKey` — documented tradeoff, acceptable for a personal tool; noted in ACTIONS.md.
- **MEDIUM:** OAuth client secret / S3 keys / FTP passwords stored plaintext in `targets.json`. Mitigated: file written 0600 via CreateTemp+rename. Keychain storage is future work.
- **LOW:** IPC socket default perms → chmod 0600 (fixed in this pass).
- **LOW:** AppleScript strings escaped via dedicated quoting helper — injection-safe for paths.
- No hardcoded secrets; no unvalidated deserialization of untrusted data (IPC is same-user only).

## Refactoring priorities (this pass)
1. **Split `app.go` by domain** (app core / grid / dropbar / bundles / ipc / settings+window) — highest readability win, zero behavior change.
2. **Remove `builtin.SaveTargetOption` global** → inject per-invocation `SaveOption` via `tasks.Config` — restores DI, makes gdrive testable.
3. **Frontend splits** — DropBarTile + RenameItemDialog out of DropBar.tsx; extract `useNativeFileDrop` / `useTargetShortcuts` hooks out of GridPanel.
4. Small fixes: zip Close error, bundle-skip logging, socket chmod, `printResult` rename, store tests, package doc for main.

## Deferred (not worth it at this size)
- Splitting `handleIPC` switch into per-command handlers.
- Keychain-backed secret storage.
- ESLint/prettier setup (tsc strict + small surface suffices for now).
