import { useEffect, useRef } from "react";
import { backend, type DropBarItem } from "@/lib/backend";
import { getDraggingDropBarItem, initNativeFileDrop, setDraggingDropBarItem } from "@/lib/dnd";

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
      onFiles(dropId, paths) {
        if (!dropId || paths.length === 0) return;
        backend.playDropSound();
        if (dropId === "dropbar") {
          backend.dropBar.add({ kind: "files", paths });
          return;
        }
        if (dropId === "add-to-grid") {
          backend.grid.addFromPaths(paths);
          return;
        }
        const isDropBarItem = itemsRef.current.some((item) => item.id === dropId);
        if (isDropBarItem) {
          // Consume the in-flight drag-out source (if any) so a stale value
          // can't leak into an unrelated later drop.
          const sourceId = getDraggingDropBarItem();
          setDraggingDropBarItem(null);
          if (sourceId && sourceId !== dropId) {
            backend.dropBar.combine(dropId, sourceId);
          } else {
            // Not a drag-out-in-progress (e.g. an external Finder file
            // dropped over an existing tile): fall back to the plain stash.
            backend.dropBar.add({ kind: "files", paths });
          }
          return;
        }
        // Like Dropzone: the grid closes right after a drop on an action;
        // the task keeps running with progress in the menu bar icon.
        backend.drop(dropId, { kind: "files", paths });
        backend.window.hide();
      },
    });
  }, []);
}
