# DragZone — File Structure Map

*Analysis date: 2026-07-05. ~8.4k lines of first-party source (excl. generated wailsjs, shadcn ui primitives).*

## Backend (Go)
| Area | Files | Notes |
|---|---|---|
| Entry / facade | `main.go`, `app.go` (671) | **FLAG: app.go mixes 7 domains** — split by domain |
| CLI | `cmd/dz/main.go` | `print_` is un-idiomatic naming |
| Domain types | `internal/model/model.go` | clean, json-tagged |
| Action engine | `internal/actions/actions.go` | **FLAG: builtin.SaveTargetOption package global** (in builtin/gdrive.go) should be DI |
| Built-in actions | `internal/actions/builtin/*` (15 files) | one action per file — good; zip.go swallows Close error |
| Script bundles | `internal/bundles/{meta,shims,action,template}.go` | LoadDir silently skips broken bundles |
| Task runner | `internal/tasks/runner.go` | positional NewRunner params → config struct |
| Persistence | `internal/{storage,config,grid,dropbar}` | atomic writes via CreateTemp (0600 — good for secrets) |
| IPC | `internal/ipc/ipc.go` | socket perms default — tighten to 0600 |
| Native bridge | `internal/platform/{bridge_darwin.{h,m,go},services_darwin.go}` | well-isolated cgo boundary |
| Utilities | `internal/fsutil/fsutil.go` | pure, tested |

## Frontend (TypeScript/React)
| Area | Files | Notes |
|---|---|---|
| Shell | `App.tsx`, `main.tsx` | clean |
| Facade | `lib/backend.ts` | single typed wrapper over wailsjs — good |
| DnD plumbing | `lib/dnd.ts` | mime constants + native drop routing |
| Live state | `hooks/useBackend.ts`, `hooks/useFileIcon.ts` | event-subscribed hooks |
| Grid feature | `features/grid/{GridPanel,TargetTile,AddTargetDialog,OptionsForm}.tsx` | **FLAG: GridPanel embeds shortcut + native-drop effects** → extract hooks |
| Drop Bar feature | `features/dropbar/{DropBar,PopoutBar}.tsx` | **FLAG: DropBar.tsx (217) holds 3 components** → split tile + rename dialog |
| Tasks feature | `features/tasks/{TaskList,InputRequestDialog}.tsx` | clean |
| Settings feature | `features/settings/SettingsDialog.tsx` | DevelopActionRow inline — acceptable, extract for symmetry |
| shadcn primitives | `components/ui/*` (19) | generated, leave untouched |

## Tests
`internal/bundles/bundles_test.go` (live python3 round-trip), `internal/fsutil/fsutil_test.go`, `internal/actions/builtin/zip_test.go`. **Gap:** no tests for grid/dropbar stores or ipc — add store tests.

No orphaned files. Naming is consistent (Go: lower_snake files; TSX: PascalCase components, camelCase libs/hooks).
