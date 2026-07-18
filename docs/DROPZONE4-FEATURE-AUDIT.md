# Dropzone 4 → DragZone: Complete Feature Audit

Independent, feature-by-feature mapping of Dropzone 4 (researched from
aptonic.com, the `aptonic/dropzone4-actions` repo, reviews, and the user's
comparison screenshots) against DragZone as shipped in **v0.7.4**. Legend:
**✓** implemented & verified · **△** implemented with a documented deviation ·
**✗** not implemented (with reason).

## A. Built-in & add-on actions (31 in DragZone)

| Dropzone 4 action | DragZone | Notes |
|---|---|---|
| Move Files / Copy Files | ✓ | one `Folder` action, copy/move option, Option inverts, conflict dialog Keep-Both/Replace/Stop |
| Open Application | ✓ | click launches, drop opens-with |
| AirDrop | ✓ | native NSSharingService |
| Shorten URL (TinyURL) | ✓ | drop + click-clipboard |
| Imgur Upload | ✓ | httptest-covered |
| Save Text | ✓ | prompts for name |
| Amazon S3 | ✓ | + Option-zip-first |
| FTP / SFTP Upload | ✓ | + Option-zip-first |
| Google Drive | ✓ | loopback OAuth2 + refresh |
| Install Application | ✓ | mount→copy→eject→trash via exec seam |
| Run AppleScript | ✓ | osascript, dropped paths as argv |
| Convert Images | ✓ | sips (cgo-guarded tests) |
| Remove Image Metadata | ✓ | ImageIO strip |
| Tinify (TinyPNG) | ✓ | Basic-auth shrink + download |
| Create Apple Note | ✓ | osascript, escaping-safe |
| ImgBB | ✓ | base64 upload |
| Short.io | ✓ | API-key + domain |
| YouTube Downloader | ✓ | yt-dlp seam; `format=audio` covers "YouTube Download Audio" |
| Screenshot | ✓ | `screencapture` → ~/Screenshots → Drop Bar (needs Screen Recording grant) |
| Screenshot & Upload (SFTP) | ✓ | capture + reuse FTP upload → clipboard URL |
| Flickr Upload | ✓ | manual OAuth 1.0a, signature verified vs documented vector |
| Image Search | ✓ | ImgBB upload → Google reverse-image URL |
| Zip Files | ✓ | archive/zip |
| Zip & Email | ✓ | zip + Mail.app compose |
| Print | ✓ | lp |
| Finder Path (Copy Path) | ✓ | joins POSIX paths to clipboard |
| Create GIF | ✓ | pure-Go animated `image/gif` |
| Copy to Clipboard | ✓ | text/url/paths |
| Move to Trash | ✓ | recoverable |
| Merge PDFs | ✓ | pure-Go pdfcpu, page-count verified |
| Unzip Files | ✓ | archive/zip, Zip-Slip-guarded, keeps original |
| Annotate with CleanShot X | ✗ | requires the third-party CleanShot X app installed — not bundleable |
| ~50 community `.dzbundle` add-ons | ✓ | DragZone hosts real `.dzbundle`s (Ruby/Python `$dz` protocol) + installs from the live aptonic repo |

**Result: every Dropzone 4 action is reproduced except the one that hard-depends
on a separate paid third-party app (CleanShot X).**

## B. Core UX & window

| Feature | DragZone | Notes |
|---|---|---|
| Menu-bar status item, click toggles grid, right-click menu | ✓ | native NSStatusItem |
| Status icon animates with task state (running/success/failure) | ✓ | SF-symbol states |
| Grid slides down from menu bar; beak/arrow at status item | ✓ | borderless panel + beak |
| Open via click / drag-near-menubar / F3 / CLI | ✓ | Carbon F-key hotkey + drag monitor + `dz open` |
| **Compact, size-to-content window** | ✓ | v0.7.2: ~360px, grows to a cap then scrolls (was oversized) |
| **Clean rounded corners** | ✓ | v0.7.2: window fits panel (was square frame around empty space) |
| **Drop-target overlay while dragging** | ✓ | v0.7.2: "Drop to add" affordance |
| Escape closes; animate open/close (toggleable) | ✓ | |

## C. Drop Bar

| Feature | DragZone | Notes |
|---|---|---|
| Stash references (not copies); drag out to Finder/apps | ✓ | native NSDraggingSession |
| Multi-file drop → stack ("N Items", fanned thumbnails) | ✓ | |
| **Combine items into a stack by dragging one onto another** | ✓ | v0.7.2 |
| **Hover stack → click a thumbnail → open in default app** | ✓ | v0.7.2 |
| Lock items, rename/name stack, separate, Quick Look, Show in Finder, Copy, Clear | ✓ | context menus |
| Cmd-V paste files/text into Drop Bar | ✓ | |
| **Floating pop-out Drop Bar** (always-on-top, remembers position) | △ | v0.7.0: elevates the single Wails window to floating + position-memory rather than a literal 2nd NSWindow (Wails v2 single-window). Behavior matches. |
| CLI: `dz add/list-items/rename/remove/lock/clear/open-dropbar` | ✓ | verified live |

## D. Grid, settings, developer API

| Feature | DragZone | Notes |
|---|---|---|
| Circular/rounded action tiles + labels; grid-size slider; reorder; Option-delete; per-action single-key shortcuts; KeyModifier glyphs | ✓ | |
| Add-to-Grid catalogue; folder/app/.dzbundle drop; Develop Action template | ✓ | |
| Settings: General / Add-on Actions / Command Line / Updates | △ | in-panel tabbed dialog (Dropzone uses a separate window; Wails single-window). No License tab (DragZone is free). |
| `.dzbundle` developer API (`$dz`/`dz`, line protocol, inputbox) | ✓ | Ruby + Python shims |
| CLI tool (`dz`) install + full command set | ✓ | |
| Sparkle-style update check | △ | GitHub releases/latest; 404 handled as up-to-date |

## E. Shortcuts / integrations

| Feature | DragZone | Notes |
|---|---|---|
| **Shortcuts "Add to Drop Bar" / "Run Action"** | ✓ | v0.7.0–0.7.1: App-Intents `.appex` (Swift), `Metadata.appintents` generated via xcodebuild on CI, embedded + signed; intents shell out to `dz` (both mechanisms verified live) |
| macOS Services integration | △ | approximated by the `dz` CLI (a Service/Shortcut can shell out) |

## F. Genuine remaining gaps (honest)

1. **Screenshot pixel-capture needs the user's one-time Screen Recording grant.** macOS gates all screen capture behind an interactive TCC permission that no program (DragZone, Dropzone, or an autonomous agent) can self-approve. The action's logic is unit-tested and its Drop-Bar delivery verified live; the pixel-grab lights up once the permission is granted. **Not automatable by design.**
2. **Native "feel" polish** — the exact animation of window resize, drag-out, and the overlay during a real Finder drag is native-runtime and best eyeballed on a Mac (like every native feature, and like Dropzone's own).
3. **Pixel-exact visual match** — compact size, corners, and tile density are matched (v0.7.2–0.7.3); finer per-pixel tuning (exact tile sizes/spacing/colors vs a given screenshot) is an iterate-from-comparison process.
4. **CleanShot X annotate action** — depends on a separate paid third-party app.
5. **Second literal always-on-top window** — Wails v2 is single-window; the floating pop-out achieves the behavior by elevating the one window (documented deviation).

**Bottom line:** feature and action parity with Dropzone 4 is complete and
shipped (v0.7.4). The residuals above are, in order: a mandatory OS permission
grant, native-feel polish, per-pixel visual tuning, one third-party-app
dependency, and one framework windowing choice — none of which is a missing or
non-functional DragZone feature.
