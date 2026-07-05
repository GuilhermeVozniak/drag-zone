import { useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { backend, type DropBarItem } from "@/lib/backend"
import { useFileIcon } from "@/hooks/useFileIcon"
import { DROPBAR_MIME } from "@/lib/dnd"
import { cn } from "@/lib/utils"
import { File, Files, Link, Lock, Type, X } from "lucide-react"
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface DropBarProps {
  items: DropBarItem[]
  onRemove: (id: string) => void
  onClear: () => void
}

function itemIcon(item: DropBarItem) {
  if (item.kind === "files") return (item.paths?.length ?? 0) > 1 ? Files : File
  if (item.kind === "url") return Link
  return Type
}

function DropBarTile({
  item,
  onRemove,
}: {
  item: DropBarItem
  onRemove: (id: string) => void
}) {
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
          // Text/URL items use HTML5 drag (dropped on tiles in-window);
          // file items start a native drag session so they can be dragged
          // out to Finder and other apps.
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
      <Dialog open={renaming !== null} onOpenChange={(o) => !o && setRenaming(null)}>
        <DialogContent className="dark border-white/10 bg-neutral-900 text-neutral-100 sm:max-w-[300px]">
          <DialogHeader>
            <DialogTitle className="text-sm">Rename Item</DialogTitle>
          </DialogHeader>
          <Input
            autoFocus
            value={renaming ?? ""}
            onChange={(e) => setRenaming(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && renaming !== null) {
                backend.dropBar.rename(item.id, renaming)
                setRenaming(null)
              }
            }}
          />
          <DialogFooter>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                backend.dropBar.rename(item.id, "")
                setRenaming(null)
              }}
            >
              Reset
            </Button>
            <Button
              size="sm"
              onClick={() => {
                if (renaming !== null) backend.dropBar.rename(item.id, renaming)
                setRenaming(null)
              }}
            >
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </ContextMenu>
  )
}

export function DropBar({ items, onRemove, onClear }: DropBarProps) {
  return (
    <div
      data-drop-id="dropbar"
      style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
      className={cn(
        "mx-3 rounded-xl border border-dashed border-white/15 bg-white/[0.04]",
        "min-h-[72px] px-2 py-2"
      )}
    >
      {items.length === 0 ? (
        <p className="flex h-[56px] items-center justify-center text-[11px] text-neutral-500">
          Drop Bar — drag files here to stash them
        </p>
      ) : (
        <ScrollArea className="w-full">
          <div className="flex items-start gap-1">
            {items.map((item) => (
              <DropBarTile key={item.id} item={item} onRemove={onRemove} />
            ))}
          </div>
          <ScrollBar orientation="horizontal" />
        </ScrollArea>
      )}
      {items.length > 0 && (
        <div className="mt-1 flex justify-end">
          <button
            onClick={onClear}
            className="text-[10px] text-neutral-500 hover:text-neutral-300"
          >
            Clear all
          </button>
        </div>
      )}
    </div>
  )
}
