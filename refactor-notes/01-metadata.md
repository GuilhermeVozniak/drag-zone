# DragZone — Project Metadata

*Analysis date: 2026-07-05 (initial audit, commit 456ffbb)*

## Project
- **Name:** dragzone — Dropzone 4 clone for macOS (menu bar drag-and-drop utility)
- **Repo:** github.com/GuilhermeVozniak/drag-zone

## Runtime & frameworks
- **Backend:** Go 1.23+ (toolchain 1.26.4), Wails v2.12.0, cgo/Objective-C bridge (Cocoa, Carbon, ServiceManagement, ImageIO)
- **Frontend:** React 19, TypeScript 5.8 (strict), Vite 6, Tailwind CSS v4, shadcn/ui (new-york), lucide-react
- **Scripting host:** system `ruby` / `python3` for .dzbundle actions

## Dependencies
- **Go direct:** wails/v2, google/uuid, x/crypto+pkg/sftp (SFTP), jlaffaye/ftp, aws-sdk-go-v2 (s3/config/credentials/manager), x/oauth2. All current; aws-sdk-go-v2 is the only heavy tree and is justified by the S3 action.
- **Frontend prod:** react, react-dom, lucide-react, clsx, tailwind-merge, class-variance-authority + radix-ui primitives pulled by shadcn components. Dev: vite, typescript, tailwind. No red flags; no deprecated packages.

## Build entry points
- `wails build` → build/bin/dragzone.app (regenerates frontend/wailsjs bindings, runs npm build)
- `go build -o build/bin/dz ./cmd/dz` → CLI companion binary
- `go test ./internal/...` — backend tests
- Gotcha: `wails` commands must run from the repo root; from frontend/ they fail with a misleading `frontend/wails.json` error.
