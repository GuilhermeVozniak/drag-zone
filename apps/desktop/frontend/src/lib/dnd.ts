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
  onFiles(dropId: string | null, paths: string[]): void;
}

/** Registers the global native file-drop listener. Call once. */
export function initNativeFileDrop(handler: DropHandler) {
  OnFileDrop((x, y, paths) => {
    document.body.classList.remove("native-dragging");
    const el = document.elementFromPoint(x / scale, y / scale);
    const dropEl = el?.closest<HTMLElement>("[data-drop-id]");
    handler.onFiles(dropEl?.dataset.dropId ?? null, paths);
  }, true);
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
