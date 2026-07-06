# Dropzone 4 UX Specification

Synthesized 2026-07-06 from 115 research findings (6 parallel investigators
over aptonic.com, reviews, release notes, and the real app's resources and
binary strings) plus user-provided screenshots. Tags: [C]onfirmed /
[L]ikely / [U]ncertain. ✓ = implemented in DragZone, △ = approximated,
✗ = not implemented.

## Menu bar icon
- [C] ✓ Template glyph: rounded square with a downward arrow into a tray
  (`tray.and.arrow.down`), NOT a grid glyph.
- [C] ✓ State images: bare down-arrow during drags, checkmark on success,
  X on failure, animated variant while tasks run. DragZone switches SF
  symbols (running/success/failure revert to normal after 2s idle).
- [C] ✓ Standard highlighted background while the grid is open.

## Window / invocation
- [C] ✓ Borderless rounded panel with an upward beak pointing at the status
  item; NOT an NSPopover. Flat, near-opaque background: light ~#ECECEC
  (~96.5% alpha), dark #303030 (~97%). ~10pt corner radius.
- [C] ✓ Opens by status-item click, drag near the menu bar, or the grid
  hotkey (default F3); Escape closes; animates open/close (toggleable).
- [C] △ Drag-initiated reveal: real app first shows a small overlay tab with
  the app icon below the status item, expanding to the grid when the drag
  reaches it. DragZone opens the grid directly when the drag nears the menu
  bar (toggleable, "Show drag target overlay").
- [L] ✓ Grid hides right after a drop on an action; the task continues with
  progress in the menu bar icon and a notification.

## Header toolbar
- [C] ✓ Left: '+' button | thin separator | double-chevron that collapses/
  expands the TOP SECTION (chevrons point up when expanded, down when
  collapsed). Center: "Recently Shared ▾" bordered pill (only once shares
  exist) listing recent upload URLs. Right: pop-out-Drop-Bar (overlapping
  rectangles) | separator | gear.
- [C] △ '+' opens a menu of built-in actions (icons + names), ending with
  "Get More Actions…" and Option-revealed "Develop Action…". DragZone shows
  all actions plus a visible Develop entry; Get More → Settings Add-ons tab.
- [C] △ Gear opens a menu: Settings… (⌘,), Action Console (⇧⌘D, hidden),
  Help, About, Quit. DragZone: Settings…, Open Add-on Actions Folder, Quit.

## Top section (Drop Bar row)
- [C] ✓ Persistent dashed rounded-square tiles: "Add to Grid" (+ glyph) and
  "Drop Bar" (solid down arrow) — both remain visible even with items.
- [C] ✓ Dropping an app on Add to Grid adds a launch tile; a folder adds a
  folder target; a .dzbundle installs the action.
- [C] ✓ Items are references; multi-file drops become stacks labeled
  "N Items"; stacks render as fanned/offset thumbnails with photo borders.
- [C] ✓ Item context menu: Lock Items, Separate Items, Name Stack (⏎),
  Quick Look (Space), Show in Finder, Copy to Clipboard, Remove.
- [C] ✓ Bar background context menu: Combine all Items to Stack, Clear.
- [C] ✓ Cmd-V pastes clipboard files/text into Drop Bar.
- [C] ✓ Drag-out removes the item unless locked (Option keeps it — DragZone
  uses lock + global setting instead of the Option modifier). △
- [C] ✓ Items wrap onto extra rows. [L] Real app scrolls with chevron
  affordance for very many items — DragZone wraps only. △
- [C] ✓ Pop-out floating Drop Bar via toolbar button (real: separate small
  window w/ close X; DragZone: same window switches to a compact pinned
  strip — single-window constraint). △ Separate F4-style hotkey ✓.

## Sections & tiles
- [C] ✓ Order: top section → hairline → "FOLDERS / APPS" → hairline →
  "ACTIONS" → "TASK PROGRESS" (only while tasks run). ALL-CAPS gray labels.
- [C] ✓ Tiles are large borderless icons (~64-70pt, GridIconSize=70 default,
  adjustable) with small gray labels; real Finder icons for folders/apps;
  raw full-color branded artwork for actions (DragZone approximates
  non-bundle actions with colored shapes + glyphs). △
- [C] ✓ Folder tiles configured to copy show a green circled '+' badge
  bottom-right; move mode shows none.
- [C] ✓ Drag-hover darkens the hovered tile's icon itself (like Finder);
  no background ring.
- [C] ✓ Hold Option → X delete badges on tiles (real: top-left gray square
  w/ white X; jiggle exists behind a default). Right-click menu has Edit /
  Duplicate and Modify / Remove (DragZone: Edit… / Remove). △
- [C] ✓ Drag to reorder tiles. Click behavior: folder opens, app launches,
  action runs click handler or opens config when it has none. △ (DragZone
  errors instead of opening config for handler-less actions.)
- [C] ✓ F3 opens grid with single-key overlays on tiles (dark rounded badge,
  white letter, ServiceKeyOverlays toggle); pressing the key runs it.
- [C] ✗ KeyModifiers overlay (⌘ glyph over action while dragging with a
  declared modifier).

## Tasks
- [C] ✓ "TASK PROGRESS" section at panel bottom: action icon + status label
  above a thin blue progress bar + circular X cancel button; determinate
  fills by percent, indeterminate animates. Cancel aborts the task.
- [C] ✓ Completion notification; TaskFinished sound (DragZone: Glass/Basso);
  GridDrop sound on drop (DragZone: Pop). Play sounds toggle.

## Settings (tabbed window; DragZone: tabbed dialog △)
- [C] ✓ General: Grid size slider, Open grid shortcut, Pop out Drop Bar
  shortcut, Always use dark mode, Animate grid, Show service key overlays;
  System: Launch at login, Notifications, Play sounds; Behaviour: drag
  target overlay toggle.
- [C] ✓ Add-on Actions tab with Install buttons (DragZone: live list from
  aptonic/dropzone4-actions; installs the real bundles).
- [C] ✓ Command Line tab: install the dz tool (/usr/local/bin) + usage.
- [C] ✓ Updates tab: auto-check toggle, Check Now, version line.
- [C] ✗ License tab (not applicable — DragZone is free).

## Appearance
- [C] ✓ Full light + dark themes; "Always use dark mode" override; branded
  icons stay colored in both.
- [C] ✗ First-run 6-slide onboarding carousel.

## Known intentional deviations
Single Wails window (pop-out bar replaces the window content instead of a
second window; settings are a dialog, not a separate window); no jiggle
animation; no Action Console; drag overlay opens the grid directly.
