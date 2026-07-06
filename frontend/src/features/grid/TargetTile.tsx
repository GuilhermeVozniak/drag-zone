import { useState } from "react"
import type { ActionSpec, Target } from "@/lib/backend"
import { useFileIcon } from "@/hooks/useFileIcon"
import { ActionTileIcon } from "@/components/ActionIcon"
import { Plus, X } from "lucide-react"
import { DROPBAR_MIME, TARGET_MIME, payloadFromDataTransfer } from "@/lib/dnd"
import { cn } from "@/lib/utils"
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"

interface TargetTileProps {
  target: Target
  spec: ActionSpec | undefined
  showKeyOverlay: boolean
  optionHeld: boolean
  onClick: () => void
  onEdit: () => void
  onRemove: () => void
  onDropBarItemDrop: (itemId: string) => void
  onTextDrop: (text: string, isUrl: boolean) => void
  onReorder: (draggedTargetId: string) => void
}

export function TargetTile({
  target,
  spec,
  showKeyOverlay,
  optionHeld,
  onClick,
  onEdit,
  onRemove,
  onDropBarItemDrop,
  onTextDrop,
  onReorder,
}: TargetTileProps) {
  const [hover, setHover] = useState(false)
  // Folder and app tiles show the real Finder icon of their configured path.
  const nativeIcon = useFileIcon(target.options?.path || target.options?.app)

  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <button
          data-drop-id={target.id}
          draggable
          onDragStart={(e) => {
            e.dataTransfer.setData(TARGET_MIME, target.id)
            e.dataTransfer.effectAllowed = "move"
          }}
          className={cn(
            "group relative flex w-[76px] flex-col items-center gap-1.5 rounded-xl p-2 outline-none",
            "transition-transform duration-100",
            hover && "scale-105"
          )}
          style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
          onClick={onClick}
          onDragOver={(e) => {
            e.preventDefault()
            setHover(true)
          }}
          onDragLeave={() => setHover(false)}
          onDrop={(e) => {
            e.preventDefault()
            setHover(false)
            const draggedTarget = e.dataTransfer.getData(TARGET_MIME)
            if (draggedTarget && draggedTarget !== target.id) {
              onReorder(draggedTarget)
              return
            }
            const itemId = e.dataTransfer.getData(DROPBAR_MIME)
            if (itemId) {
              onDropBarItemDrop(itemId)
              return
            }
            const payload = payloadFromDataTransfer(e.dataTransfer)
            if (payload?.text) onTextDrop(payload.text, payload.kind === "url")
          }}
        >
          <div
            className={cn(
              "flex size-[52px] items-center justify-center rounded-xl",
              "transition-all duration-100",
              // A dragged file darkens the hovered icon, like Finder's
              // drop-target folders — no ring or background.
              hover && "brightness-[0.6] saturate-150"
            )}
          >
            {nativeIcon ? (
              <img
                src={`data:image/png;base64,${nativeIcon}`}
                alt=""
                className="size-[52px]"
                draggable={false}
              />
            ) : (
              <ActionTileIcon
                actionId={target.actionId}
                icon={spec?.icon}
                className="size-[46px]"
              />
            )}
          </div>
          <span className="line-clamp-2 w-full text-center text-[11px] leading-tight text-neutral-300">
            {target.label}
          </span>
          {target.shortcut && showKeyOverlay && (
            <span className="absolute left-1/2 top-5 z-10 -translate-x-1/2 rounded-md bg-black/60 px-1.5 py-0.5 font-mono text-[12px] font-semibold text-white">
              {target.shortcut.toUpperCase()}
            </span>
          )}
          {optionHeld && (
            <span
              role="button"
              onClick={(e) => {
                e.stopPropagation()
                onRemove()
              }}
              className="absolute left-0.5 top-0.5 z-10 flex size-5 items-center justify-center rounded-md bg-neutral-600/90 shadow"
              title="Remove from Grid"
            >
              <X className="size-3 text-white" />
            </span>
          )}
          {target.actionId === "folder" && target.options?.mode === "copy" && (
            <span className="absolute bottom-5 right-2 z-10 flex size-4 items-center justify-center rounded-full bg-green-500 shadow">
              <Plus className="size-3 text-white" strokeWidth={3} />
            </span>
          )}
        </button>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={onEdit}>Edit…</ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem variant="destructive" onClick={onRemove}>
          Remove from Grid
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  )
}
