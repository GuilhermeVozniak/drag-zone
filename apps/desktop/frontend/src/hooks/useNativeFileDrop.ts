import { useEffect } from "react"
import { backend } from "@/lib/backend"
import { initNativeFileDrop } from "@/lib/dnd"

/**
 * Routes native (Wails) file drops to whatever is under the cursor: the
 * Drop Bar stashes the files, a target tile runs its action. Register once
 * per window.
 */
export function useNativeFileDrop() {
  useEffect(() => {
    initNativeFileDrop({
      onFiles(dropId, paths) {
        if (!dropId || paths.length === 0) return
        backend.playDropSound()
        if (dropId === "dropbar") {
          backend.dropBar.add({ kind: "files", paths })
        } else if (dropId === "add-to-grid") {
          backend.grid.addFromPaths(paths)
        } else {
          // Like Dropzone: the grid closes right after a drop on an action;
          // the task keeps running with progress in the menu bar icon.
          backend.drop(dropId, { kind: "files", paths })
          backend.window.hide()
        }
      },
    })
  }, [])
}
