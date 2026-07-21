import { useEffect, useRef } from "react";
import { backend, type DropBarItem, events } from "@/lib/backend";
import {
  getDraggingDropBarItem,
  initNativeFileDrop,
  reorderIndex,
  setDraggingDropBarItem,
} from "@/lib/dnd";
import { reportError } from "@/lib/report";

/**
 * Routes native (Wails) file drops to whatever is under the cursor: the
 * Drop Bar stashes the files, a target tile runs its action, and dropping
 * onto another Drop Bar tile mid drag-out combines the two into a stack.
 * Register once per window; pass the live Drop Bar items so drops onto an
 * item tile can be told apart from drops onto the bar's background.
 */
export function useNativeFileDrop(dropBarItems: DropBarItem[] = []) {
  // A ref, not a dependency: the listener registers once, but always reads
  // the latest item list without re-subscribing to OnFileDrop on every
  // Drop Bar change.
  const itemsRef = useRef(dropBarItems);
  itemsRef.current = dropBarItems;

  useEffect(() => {
    initNativeFileDrop({
      onFiles(dropId, paths, zone) {
        // Capture the in-flight drag-out source (if any) and clear the
        // global immediately, before any branching below, so it is valid
        // for at most this one drop. Without this, a drag-out that resolves
        // anywhere other than a sibling Drop Bar tile — Finder (the common
        // case: OnFileDrop never fires for out-of-window drops), the Drop
        // Bar background, add-to-grid, a grid target, or a cancelled drag —
        // would leave the tracker set, and a later unrelated drop onto a
        // Drop Bar tile would misread it as "combine stale source into this
        // tile", discarding the newly dropped file. See also the
        // drag-session-end subscription below, which clears the tracker
        // even when no drop ever lands (out-of-window drops, cancels).
        const sourceId = getDraggingDropBarItem();
        setDraggingDropBarItem(null);

        if (!dropId || paths.length === 0) return;
        backend.playDropSound();
        if (dropId === "dropbar") {
          backend.dropBar
            .add({ kind: "files", paths })
            .catch((err) => reportError("Couldn't stash", err));
          return;
        }
        if (dropId === "add-to-grid") {
          backend.grid.addFromPaths(paths).catch((err) => reportError("Couldn't add", err));
          return;
        }
        const isDropBarItem = itemsRef.current.some((item) => item.id === dropId);
        if (isDropBarItem) {
          if (sourceId && sourceId !== dropId) {
            // Landing on a tile's edge reorders next to it; landing on the
            // center combines into a stack (Dropzone's behavior).
            if (zone !== "center") {
              const idx = reorderIndex(itemsRef.current, sourceId, dropId, zone === "after");
              if (idx != null) {
                backend.dropBar
                  .move(sourceId, idx)
                  .catch((err) => reportError("Couldn't reorder", err));
                return;
              }
            }
            backend.dropBar
              .combine(dropId, sourceId)
              .catch((err) => reportError("Couldn't combine", err));
          } else {
            // Not a drag-out-in-progress (e.g. an external Finder file
            // dropped over an existing tile): fall back to the plain stash.
            backend.dropBar
              .add({ kind: "files", paths })
              .catch((err) => reportError("Couldn't stash", err));
          }
          return;
        }
        // Like Dropzone: the grid closes right after a drop on an action;
        // the task keeps running with progress in the menu bar icon. On
        // failure the window stays open so the error toast is seen.
        backend
          .drop(dropId, { kind: "files", paths })
          .then(() => backend.window.hide())
          .catch((err) => reportError("Drop failed", err));
      },
    });
    // Belt-and-suspenders: the Go side emits this after every drag-out
    // session ends, whatever the outcome, so the tracker never survives a
    // drag that resolves without ever reaching onFiles above (e.g. dropped
    // outside the window onto Finder, or cancelled).
    return events.onDropBarDragEnded(() => setDraggingDropBarItem(null));
  }, []);
}
