# Dropzone 4 — Screenshot Handling Research + DragZone Gap Map

Researched 2026-07-18 from aptonic.com blog, Dropzone forums, reviews, and the
community `dropzone4-actions` repo. Goal: reproduce Dropzone 4's screenshot
experience (and close remaining parity gaps) in DragZone.

## How Dropzone 4 actually does screenshots

There is **no single “native screenshot capture” in Dropzone core.** The
screenshot experience is assembled from three mechanisms:

1. **Screenshot Action (`screencapture` + Drop Bar)** — the canonical one.
   A grid action runs macOS `screencapture` (interactive region select),
   saves the image to `~/Screenshots/` with a **timestamped filename**
   (`Screen Shot 2018-04-08 at 02.30.45 PM.png` — timestamped to avoid
   overwriting/duplicate Drop Bar icons), then adds it to the **Drop Bar**
   via the `$dz.add_dropbar` script API, ready to drag into any app / onto
   another action. Supports KeyModifiers (Command/Option/Control/Shift) to
   vary capture mode. (Source: Dropzone forums “Screenshot tool” thread;
   `$dz.add_dropbar` API.)
2. **Shortcuts integration** — an “Add to Drop Bar” Shortcuts action (any
   Shortcut that outputs files can push them into the Drop Bar), and a
   “take interactive screenshot → upload via SFTP → URL to clipboard”
   Shortcut that runs a preconfigured Dropzone SFTP action. Requires the
   **non-MAS build** (sandbox). (Source: aptonic.com blog, 2 posts.)
3. **Manual drop** — drag any existing screenshot file into the Drop Bar.

The **essential UX** users mean by “screenshots”: press/click → interactive
region capture → the shot instantly appears in the Drop Bar as a thumbnail,
ready to drag out or upload. Optionally the shot's URL is copied after an
upload action.

## What DragZone has today

| Piece | DragZone status |
|---|---|
| Drop Bar (stash/stack/rename/lock/drag-out/Quick Look) | ✅ full |
| `$dz.add_dropbar` for **bundle scripts** | ✅ wired (`bundles/action.go` `AddDropBar`) |
| **Built-in actions adding to Drop Bar** | ❌ `Services` has no `AddDropBar` |
| **Screenshot capture (any mechanism)** | ❌ none — no `screencapture`, no Screenshot action, no capture hotkey |
| `dz add <file>` CLI (a Shortcut can call this) | ✅ exists (approximates the Shortcuts “Add to Drop Bar”) |
| Screenshots save folder / timestamped naming | ❌ none |

**This is the “very different” gap:** DragZone can only hold screenshot files
you manually drop; it cannot *take* one. Dropzone's one-gesture capture→Drop
Bar flow is entirely absent.

## Reproduction plan (screenshots)

Build the capture side and wire it to the existing Drop Bar:

1. **`Services.AddDropBar(paths []string)`** — add to the `actions.Services`
   interface; implement in `services_darwin.go` → `App.DropBarAdd`. Lets
   built-in actions push files into the Drop Bar (parity with the script
   API). Also add `Services.CaptureScreenshot(mode, dst) error` that execs
   `screencapture` (no cgo needed — standard CLI, like `print` execs `lp`).
2. **Built-in `Screenshot` action** (`internal/actions/builtin/screenshot.go`)
   — a `Clicker`: click captures. Options: **mode** (interactive region /
   window / full screen), **save folder** (default `~/Screenshots`,
   created if absent), **after capture** (Add to Drop Bar [default] / Copy
   file to clipboard / Reveal in Finder). Timestamped filenames
   (`Screenshot YYYY-MM-DD at HH.MM.SS.png`). KeyModifiers vary the mode
   like Dropzone. Registered in `builtin.go`; auto-appears as a grid tile.
3. **Capture-to-Drop-Bar hotkey / menu** (Dropzone parity, stretch) — a
   global hotkey and a menu-bar item that runs interactive capture straight
   into the Drop Bar without opening the grid. Reuses slot mechanism in
   `dz_set_hotkey_f`.
4. **Tests** — exec seam (`var runCmd`) so the action's `screencapture`
   command construction (flags per mode, timestamped dst, save-dir creation,
   AddDropBar/clipboard/reveal branch) is unit-tested without a real capture;
   a recording `Services` fake asserts `AddDropBar` is called with the shot.
5. **Verify** — drive the built app; confirm a full-screen capture writes a
   timestamped PNG and it appears in the Drop Bar.

## Remaining parity gaps (from docs/UX-SPEC.md, to close after screenshots)

UX-SPEC marks most of the grid/drop-bar/settings UX ✅. Genuine remaining
deltas to drive toward 100% parity:
- **Screenshot action** (this doc) — the big one.
- **Missing built-in actions** vs Dropzone's later 4.x set: Tinify (TinyPNG),
  Create Apple Note, ImgBB, Short.io, YouTube Downloader, Screenshot+SFTP,
  Flickr, Zip & Email, Image Search, Finder Path. DragZone has 16 of ~26.
- **Shortcuts / App Intents integration** (“Add to Drop Bar”, “Run Action”) —
  large (needs an App Intents extension); `dz` CLI approximates it today.
- **△ approximations** to tighten: pop-out Drop Bar as a true separate
  always-on-top window (Wails single-window today); branded action artwork;
  Option-keeps-item-in-Drop-Bar modifier (uses lock+setting today).

Execution order: **screenshots first** (explicit ask + biggest experiential
gap), then the missing high-value actions, then the integration/approximation
tightening.
