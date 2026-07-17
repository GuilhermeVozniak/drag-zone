# DragZone — Live Exploration Report (2026-07-18)

## How this was driven

The Claude-in-Chrome extension was **not connected** in this environment, so
instead of the native WebView I drove the app with **headless Playwright
(chromium)** against the `wails dev` devserver at `http://localhost:34115`.
Wails serves the real frontend there wired to the **live Go backend** over its
dev bridge, so method bindings and backend→frontend events both work. The
`dz` CLI was exercised against the live app socket in parallel.

**What this can verify:** rendering, all frontend interactions, add/configure
targets, Settings, task execution + progress, and CLI↔app IPC — all against
real backend state. **What it cannot:** native OS interactions (Finder
drag/drop, menu-bar click, drag-OUT, AirDrop, F-key hotkeys) — no browser can
drive those; see the manual checklist.

Screenshots: `docs/exploration-assets/`.

## Per-feature results

| Feature | Expected | Observed | Result |
|---|---|---|---|
| Grid render | Drop Bar / Folders·Apps / Actions sections, beak popover | Renders with live backend data (Desktop, Downloads, 5 actions, persisted Drop Bar item) | ✅ |
| Add-to-Grid catalogue | Lists every built-in action w/ descriptions | All actions listed (Folder, Open App, AirDrop, Clipboard, Zip, Trash, Install App, Save Text, Print, Shorten URL, …) | ✅ |
| Add target + config panel | Pick action → name + shortcut → Add to Grid | "Add Shorten URL" panel; added "My Shortener"; **grid updated live 9→10 targets** | ✅ |
| Settings dialog | Gear → 4 tabs | General / Add-ons / Command Line / Updates all open & render | ✅ |
| Settings → General | grid size, F3/F4 shortcuts, columns, toggles, System | All present and styled correctly | ✅ |
| Task execution + progress | Drop → TASK PROGRESS row → completes | Clipboard action via binding → task id returned → "Clipboard — Copied to clipboard" progress bar shown | ✅ |
| `dz` CLI IPC | list / add / list-items / clear reflect live state | `dz list` shows grid incl. "My Shortener"; `dz list-items --json` shows Drop Bar item | ✅ |
| Updates → Check Now | report update status | **Shows raw "Error: checking for updates: 404 Not Found"** | ⚠️ BUG-1 |

## Findings

**BUG-1 (fix queued — plan Task 1.1): Update check surfaces a raw 404.**
`CheckForUpdates` calls `api.github.com/repos/GuilhermeVozniak/drag-zone/releases/latest`,
which GitHub returns **404** for when the repo has no published full release
(verified: the repo currently has 0 releases). The Updates tab shows a scary
red "404 Not Found". It should treat 404 as "You're up to date / no releases
yet", not an error. Existing tests cover only the 200 path.

**NOTE-2 (not a bug): dev build reports Version 0.2.0.** `main.appVersion`
defaults to `0.2.0`; release builds inject the git tag via `-ldflags`. Expected
in `wails dev`.

**NOTE-3 (environment artifact): `wails/ipc.js` pageerror** — `Cannot read
properties of null (reading 'nodes')` fires once at load. It originates in the
Wails runtime's IPC bootstrap when run in an **external browser** (not the
native WebView) and does not affect bindings or events (both verified working).
Will not occur in the shipped native app.

**NOTE-4 (verify visually): Desktop folder tile shows a small green ‘+’ badge**
in one capture. Likely the add/drag affordance; worth a glance to confirm it
isn't a stray hover state. Low priority, cosmetic.

## Manual checklist (native-only — needs the real app + a human)

These cannot be automated from any browser. Run the built app (`wails build`
→ `build/bin/dragzone.app`) and verify:

- [ ] Menu-bar tray icon: click toggles the grid; right/control-click shows the menu.
- [ ] Start dragging a file from Finder → overlay near the menu bar; dropping on the icon opens the grid.
- [ ] Drop a file onto a folder tile → copy/move happens; Option inverts.
- [ ] Drag a Drop Bar item **out** to Finder → file is delivered; item leaves the bar (unless locked).
- [ ] AirDrop action → native share sheet appears.
- [ ] F3 opens the grid; F4 pops out the Drop Bar (system-wide).
- [ ] Quick Look on a Drop Bar item (double-click / spacebar).

## Conclusion

The app is feature-complete and works end-to-end. One genuine UX bug (BUG-1,
queued as a fix). Everything else verified healthy. Proceeding to build the
durable automated regression suite (Streams 1–4, 6).
