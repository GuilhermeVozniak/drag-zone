// Drag-and-drop plumbing.
//
// Native file drags are delivered by the Wails runtime (OnFileDrop) with
// screen coordinates; we resolve the tile under the cursor ourselves via
// data-drop-id attributes. Internal drags (Drop Bar items onto tiles) use
// HTML5 DnD with a custom mime type.

import type { Payload } from "@/lib/backend";
import { OnFileDrop } from "../../wailsjs/runtime/runtime";

export const DROPBAR_MIME = "application/x-dragzone-dropbar-item";
export const TARGET_MIME = "application/x-dragzone-target";

// The whole UI is scaled with CSS zoom (grid size setting); native drop
// coordinates arrive in window pixels and must be un-zoomed before hit
// testing.
let scale = 1;
export function setUIScale(s: number) {
  scale = s;
}

// Tracks the Drop Bar item currently mid a native drag-out session (see
// DropBarTile's mousedown+move -> backend.dragOut). A native drag that never
// leaves the window and lands back on a sibling Drop Bar tile still delivers
// through the same OnFileDrop path as an external Finder drop (WebKit
// forwards NSDraggingDestination hits as ordinary web file drops); tracking
// the source here lets useNativeFileDrop tell "combine onto this tile" apart
// from "stash these files as a new item".
let draggingDropBarItemId: string | null = null;
export function setDraggingDropBarItem(id: string | null) {
  draggingDropBarItemId = id;
}
export function getDraggingDropBarItem(): string | null {
  return draggingDropBarItemId;
}

export interface DropHandler {
  /** Called with the drop-id of the element under the cursor (or null). */
  onFiles(dropId: string | null, paths: string[], zone: DropZone): void;
}

/**
 * Where within the target tile the cursor landed: the outer 30% edges mean
 * "reorder next to this tile", the center means "act on this tile" (combine
 * for Drop Bar items, run the action for grid tiles).
 */
export type DropZone = "before" | "after" | "center";

/** Registers the global native file-drop listener. Call once. */
export function initNativeFileDrop(handler: DropHandler) {
  // Wails ignores OnFileDrop re-registration (its internal `registered`
  // guard), and this hook's host remounts on every window show — so the
  // runtime listener is attached exactly once and always dispatches to the
  // latest handler. Without the slot, the first mount's stale closure (and
  // its frozen Drop Bar item list) would handle drops forever.
  currentHandler = handler;
  if (registered) return;
  registered = true;
  OnFileDrop((x, y, paths) => {
    const el = document.elementFromPoint(x / scale, y / scale);
    const dropEl = el?.closest<HTMLElement>("[data-drop-id]");
    currentHandler?.onFiles(dropEl?.dataset.dropId ?? null, paths, dropZone(dropEl, x / scale));
  }, true);
}

function dropZone(dropEl: HTMLElement | null | undefined, x: number): DropZone {
  if (!dropEl) return "center";
  const rect = dropEl.getBoundingClientRect();
  if (rect.width <= 0) return "center";
  const rel = (x - rect.left) / rect.width;
  if (rel < 0.3) return "before";
  if (rel > 0.7) return "after";
  return "center";
}

/**
 * Index to pass to dropBar.move so sourceId lands just before (or after)
 * targetId. The backend interprets the index post-removal (like grid.Move).
 * Returns null when either id is unknown or they're the same item.
 */
export function reorderIndex(
  items: { id: string }[],
  sourceId: string,
  targetId: string,
  after: boolean,
): number | null {
  if (sourceId === targetId) return null;
  const sourceIdx = items.findIndex((i) => i.id === sourceId);
  const targetIdx = items.findIndex((i) => i.id === targetId);
  if (sourceIdx < 0 || targetIdx < 0) return null;
  let idx = targetIdx + (after ? 1 : 0);
  if (sourceIdx < idx) idx -= 1;
  return idx;
}

let currentHandler: DropHandler | null = null;
let registered = false;

/** Test-only: clears the registration guard so each test re-registers. */
export function __resetNativeFileDropForTests() {
  currentHandler = null;
  registered = false;
}

/** Extracts a payload from an HTML5 drop event (text/URL drags, not files). */
export function payloadFromDataTransfer(dt: DataTransfer): Payload | null {
  const uri = dt.getData("text/uri-list");
  if (uri) {
    const first = uri.split("\n").find((l) => l && !l.startsWith("#"));
    if (first) return { kind: "url", text: first.trim() };
  }
  const text = dt.getData("text/plain");
  if (text) return { kind: "text", text };
  return null;
}
