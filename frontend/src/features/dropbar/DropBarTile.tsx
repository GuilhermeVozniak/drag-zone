import { useRef, useState } from "react"
import { backend, type DropBarItem } from "@/lib/backend"
import { useFileIcon } from "@/hooks/useFileIcon"
import { DROPBAR_MIME } from "@/lib/dnd"
import { File, Files, Link, Lock, Type, X } from "lucide-react"
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"
import { RenameItemDialog } from "./RenameItemDialog"

function itemIcon(item: DropBarItem) {
  if (item.kind === "files") return (item.paths?.length ?? 0) > 1 ? Files : File
  if (item.kind === "url") return Link
  return Type
}

interface DropBarTileProps {
  item: DropBarItem
  onRemove: (id: string) => void
}

/**
 * One stashed item (or stack) in the Drop Bar. File items start a native
 * drag session on drag-out so they can land in Finder and other apps;
 * text/URL items use HTML5 drag for in-window drops onto grid tiles.
 */
export function DropBarTile({ item, onRemove }: DropBarTileProps) {
  const Icon = itemIcon(item)
  const count = item.paths?.length ?? 0
  const nativeIcon = useFileIcon(item.paths?.[0])
  const dragStart = useRef<{ x: number; y: number } | null>(null)
  const isFiles = item.kind === "files"
  const [renaming, setRenaming] = useState<string | null>(null)

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <div
          draggable={!isFiles}
          onDragStart={(e) => {
            e.dataTransfer.setData(DROPBAR_MIME, item.id)
            e.dataTransfer.effectAllowed = "copyMove"
          }}
          onMouseDown={(e) => {
            if (isFiles && e.button === 0) {
              dragStart.current = { x: e.clientX, y: e.clientY }
            }
          }}
          onMouseMove={(e) => {
            const start = dragStart.current
            if (!start) return
            if (Math.hypot(e.clientX - start.x, e.clientY - start.y) > 5) {
              dragStart.current = null
              backend.dragOut(item.id)
            }
          }}
          onMouseUp={() => {
            dragStart.current = null
          }}
          className="group relative flex w-[64px] cursor-grab flex-col items-center gap-1 rounded-lg p-1.5 hover:bg-white/[0.08]"
        >
          <div className="relative flex size-10 items-center justify-center rounded-lg border border-white/10 bg-white/[0.07]">
            {nativeIcon ? (
              <img
                src={`data:image/png;base64,${nativeIcon}`}
                alt=""
                className="size-8"
                draggable={false}
              />
            ) : (
              <Icon className="size-5 text-neutral-200" strokeWidth={1.75} />
            )}
            {count > 1 && (
              <span className="absolute -right-1.5 -top-1.5 rounded-full bg-sky-500 px-1 text-[9px] font-semibold text-white">
                {count}
              </span>
            )}
            {item.locked && (
              <span className="absolute -bottom-1 -right-1 rounded-full bg-neutral-700 p-0.5">
                <Lock className="size-2.5 text-amber-400" />
              </span>
            )}
          </div>
          <span className="w-full truncate text-center text-[10px] text-neutral-400">
            {item.label}
          </span>
          <button
            onClick={() => onRemove(item.id)}
            className="absolute -left-1 -top-1 hidden rounded-full bg-neutral-700 p-0.5 group-hover:block"
          >
            <X className="size-2.5 text-white" />
          </button>
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        {isFiles && (
          <ContextMenuItem onClick={() => backend.quickLook(item.paths ?? [])}>
            Quick Look
          </ContextMenuItem>
        )}
        <ContextMenuItem
          onClick={() => backend.dropBar.setLocked(item.id, !item.locked)}
        >
          {item.locked ? "Unlock Items" : "Lock Items"}
        </ContextMenuItem>
        <ContextMenuItem onClick={() => setRenaming(item.label)}>
          Rename…
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem variant="destructive" onClick={() => onRemove(item.id)}>
          Remove
        </ContextMenuItem>
      </ContextMenuContent>
      <RenameItemDialog item={item} value={renaming} onValueChange={setRenaming} />
    </ContextMenu>
  )
}
