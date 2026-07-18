# Dropzone 4 — Feature Parity Reference

## Parity status (updated 2026-07-18)

**Built-in actions: 29 — Dropzone 4's COMPLETE built-in set.** Added this
session: Screenshot, Copy Path, Create Apple Note, ImgBB, Short.io, Tinify,
Zip & Email, Create GIF (pure-Go animated), YouTube Downloader (`yt-dlp`),
Image Search, Run AppleScript, Screenshot & Upload (SFTP), and Flickr Upload
(manual OAuth 1.0a HMAC-SHA1 signing, signature verified against the
documented Twitter vector). Every action is unit-tested (httptest / exec &
network seams) and independently reviewed.

**Pop-out Drop Bar — now floats.** The popped-out Drop Bar is elevated to a
floating always-on-top window (`NSFloatingWindowLevel`), stays visible when the
app deactivates, joins all Spaces, and remembers its position across launches
(`setFrameAutosaveName`), gated so the normal grid keeps its status-item
anchoring. This matches Dropzone 4's floating-Drop-Bar *behavior* within Wails
v2's single-window model (it elevates the one window in pop-out mode rather
than opening a literal second window — a user-invisible internal detail).
Native window behavior; verify on a real Mac like the other native features.

**Remaining delta to a pixel-identical clone — ONE item:**
- **Shortcuts / App Intents integration** — native “Add to Drop Bar” / “Run
  Action” Shortcuts actions require an App Intents *extension target* (Swift +
  `xcodebuild` + embedded `.appex` + its own signing) that the Wails build
  pipeline structurally cannot produce. The `dz add` / `dz run` CLI provides
  the same capability today (a macOS Shortcut can `Run Shell Script: dz add
  $file`), so the workflow is available — just not as a first-class action in
  the Shortcuts picker. This is a build-pipeline limitation, not a missing
  feature.

---

### (superseded) Earlier pass notes

**Built-in actions: 26** (was 16). Delivered this pass toward parity:
- ✅ **Screenshot** — native `screencapture` (interactive/window/screen) →
  timestamped file in `~/Screenshots` → straight into the Drop Bar
  (`Invocation.AddDropBar`). This is the Dropzone "screenshot experience"
  that was entirely missing. Real pixel capture needs macOS Screen Recording
  permission (grant it once to the app); action + wiring are unit-tested.
- ✅ **Copy Path** (Finder Path), **Create Apple Note** (osascript),
  **ImgBB**, **Short.io**, **Tinify (TinyPNG)**, **Zip & Email**,
  **Create GIF** (pure-Go `image/gif`, animated), **YouTube Downloader**
  (`yt-dlp`), **Image Search** (ImgBB upload → Google reverse search).

**Remaining gaps toward 100% parity (action-level parity is now essentially complete):**
- Niche/low-value actions still absent: Flickr (OAuth app), Screenshot &
  Upload (SFTP) — a Shortcut-style composite of the existing Screenshot + FTP.
- **Shortcuts / App Intents integration** ("Add to Drop Bar", "Run Action"):
  large — requires an App Intents extension. The `dz add`/`dz run` CLI
  approximates it today (a Shortcut can "Run Shell Script: dz add $file").
- **△ architectural approximations** (Wails v2 constraints): the pop-out Drop
  Bar reuses the single window instead of a separate always-on-top window;
  action artwork is generated (colored shape + glyph) rather than Dropzone's
  branded PNGs. These are deliberate, documented deviations.

---


Research compiled 2026-07-05 from aptonic.com, the `aptonic/dropzone4-actions` GitHub repo, Mac App Store, and reviews (Macworld, MacSources, Mause). This is the spec DragZone is built against.

## 1. Core UX flow

- Menu bar (status item) app; icon **animates with aggregate task progress** and highlights while the grid is open. Right-click shows a context menu (Settings, etc.).
- Ways to open the grid:
  1. Click the menu bar icon — grid **slides down** from the menu bar.
  2. **Start dragging a file** — a subtle overlay appears at the top of the screen; dragging onto the menu bar icon/overlay opens the grid; it stays open while a drag is in flight.
  3. Keyboard shortcut — **F3 default**, user-configurable, system-wide.
  4. CLI — `dz open` / `dz close`.
- Grid layout, top → bottom: **(1) Drop Bar, (2) Folders & Apps, (3) Actions**.
- **Add to Grid** area: white plus icon at top-left; drop target for folders/apps/bundles; click opens the action catalogue. Hold Option → “Develop Action…”.
- Every tile is both a **drop target** and **clickable** (per-action `Events`): click app = launch, click folder = open in Finder, click action = `clicked` event.
- **Multi-tasking**: concurrent tasks each show a labeled progress bar row in the grid; Notification Center notification + optional sound on completion.
- Text and URLs can be dropped, not just files.

## 2. Built-in actions

Free: **Move Files** (rsync-based; conflict dialog Replace/Stop/Keep Both), **Copy Files** (same), **Open Application** (drop = open-with, click = launch), **AirDrop**, **Shorten URL** (TinyURL; global hotkey Ctrl+Option+Cmd+S shortens selected URL → clipboard), **Imgur Upload** (URL → clipboard), **Save Text** (drop text → asks name → saves file).

Pro: **Amazon S3** (Option-drop = zip first; URL → clipboard), **FTP Upload** (SFTP too; many instances; Option-drop = zip first), **Google Drive** (OAuth, share link), **Install Application** (mount .dmg → copy .app to /Applications → launch → eject → trash dmg), **Run AppleScript**, per-action keyboard shortcuts.

Later 4.x additions: Convert Images (WebP etc., preset resize), Remove Image Metadata, Tinify (TinyPNG API key), Create Apple Note, ImgBB, Short.io, YouTube Downloader, Screenshot+SFTP.

Legacy/add-on: Flickr Upload, Zip & Email, Print, Image Search, Finder Path. ~50 community bundles in `aptonic/dropzone4-actions` (Slack, Gist, Pushover, SCP Upload, Make GIF, Open VS Code at Path, …). Installing add-ons is Pro.

## 3. Drop Bar

- Temporary shelf at top of grid; holds **references, not copies**.
- Drag files on to stash; multiple items dropped together merge into a **stack** (drag items onto each other to combine). Stacks can be renamed (right-click).
- **Drag-out** to Finder/apps/grid tiles; items **leave Drop Bar after drag-out by default**; right-click → **Lock Items** keeps them. Reorder by dragging within the bar. Quick Look preview supported.
- Clearing: right-click → clear/remove; CLI `dz clear` / `dz remove INDEX`.
- **Floating Drop Bar** (4.80.7x): “Pop Out Drop Bar” button → compact always-on-top window mirroring Drop Bar, accepts drags, remembers position. CLI `dz open-dropbar`/`close-dropbar`.

## 4. Grid customization

- Add targets by dragging folder/app/.dzbundle onto Add to Grid, or via plus icon → catalogue → config panel (skipped when `SkipConfig: Yes`).
- Drag tiles to reorder; multiple instances of the same action allowed; **hold Option to delete** (also right-click → remove); right-click → edit options; right-click scripted action → “Copy and Edit Script”.
- Folder tiles: configured **Move or Copy**; **Option-drop inverts**; click opens in Finder; tile shows the folder’s icon.
- Circular icon tiles with label beneath. Grid size adjustable. Single grid in v4 (multiple grids = v5).

## 5. Settings

Gear icon in grid. Tabs: **General**, **Add-on Actions**, **Command Line** (direct builds), **Updates** (Sparkle). Options: toggle-grid shortcut (default F3), grid size, notifications on/off, sound on/off, launch at login, per-action shortcuts (Pro), Subscribe/trial.

## 6. Developer API (.dzbundle)

- Bundle dir: `action.rb` (Ruby) or `action.py` (Python) + `icon.png` + resources. Installed to `~/Library/Application Support/Dropzone 4/Actions/`. **No action.json** — metadata is a comment header at the top of the script:

```ruby
# Dropzone Action Info
# Name: My Action
# Description: Does something useful
# Handles: Files            (Files | Text | Files, Text)
# Creator: Your Name
# URL: https://example.com
# Events: Dragged, Clicked  (optional; default both)
# KeyModifiers: Command, Option   (held modifier → ENV['KEY_MODIFIERS'])
# OptionsNIB: Login          (Login | ExtendedLogin | APIKey | UsernameAPIKey | ChooseFolder | ChooseApplication)
# OptionsTitle: Service Login Details
# SkipConfig: No
# RunsSandboxed: Yes
# UseSelectedItemNameAndIcon: No
# Version: 1.0
# UniqueID: 1234567890
# MinDropzoneVersion: 4.0
```

- Entry points `dragged` / `clicked`. Inputs: Ruby `$items` / Python `items` (file paths, or `[0]` = text). `ENV['dragged_type']` = `files`|`text`. Options/saved values arrive as env vars.
- `$dz` methods: `begin(msg)` (create/update task row), `determinate(bool)`, `percent(0-100)`, `finish(msg)` (notification; task ends only after a following `url`/`text`), `url(url[, title])` / `url(false)`, `text(text)` (clipboard + end), `fail(msg)`, `error(title,msg)` (modal, terminates), `alert(title,msg)`, `inputbox(title,prompt[,field])`, `pashua(config)`, `save_value(name,val)` / `remove_value(name)` (later exposed as env), `read_clipboard`, `temp_folder`, `add_dropbar(items)`.
- Protocol is line-based over stdout (`puts`/`print`). Debug console shows raw protocol; auto-opens on script exceptions.
- Authoring: plus icon + Option → “Develop Action…” generates a template bundle; `UniqueID` powers update checks.

## 7. Misc & Free-vs-Pro

- Clipboard is the output bus for upload/shorten actions.
- Option-key summary: invert folder move/copy; zip-before-upload (FTP/S3); delete grid items; Develop Action…; pop out floating Drop Bar.
- Shortcuts app + macOS Services integration (Pro).
- CLI (`dz`, 4.80.49+): `list`, `run NAME dragged|clicked [FILES…]`, `list-items [--json]`, `add [--stack] FILES…`, `rename INDEX NAME|--reset`, `remove INDEX`, `lock/unlock INDEX`, `clear`, `open`, `close`, `open-dropbar`, `close-dropbar`.
- Free: Drop Bar, folders/apps, Move/Copy, Open App, AirDrop, Imgur, limited TinyURL. Pro: Google Drive, S3, FTP/SFTP, unlimited shortening, AppleScript/services, per-action shortcuts, add-on library, custom actions. (Clone: everything free.)
- Requirements drifted to macOS 11+; Intel + Apple Silicon.
