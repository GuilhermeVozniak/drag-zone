import type { DropBarItem } from "@/lib/backend"
import { backend } from "@/lib/backend"
import { ArrowDownToLine, Plus } from "lucide-react"
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"
import { DropBarTile } from "./DropBarTile"

interface TopSectionProps {
  items: DropBarItem[]
  /** The Add to Grid tile is omitted in the popped-out Drop Bar. */
  showAddToGrid?: boolean
  onAddClick?: () => void
}

function DashedTile({
  dropId,
  label,
  onClick,
  children,
}: {
  dropId: string
  label: string
  onClick?: () => void
  children: React.ReactNode
}) {
  return (
    <button
      data-drop-id={dropId}
      onClick={onClick}
      style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
      className="flex w-[76px] flex-col items-center gap-1.5"
    >
      <span className="flex size-[52px] items-center justify-center rounded-xl border-2 border-dashed border-neutral-500 transition-colors hover:border-neutral-300">
        {children}
      </span>
      <span className="text-[11px] text-neutral-400">{label}</span>
    </button>
  )
}

/**
 * The grid's top section, like Dropzone 4: the persistent dashed "Add to
 * Grid" and "Drop Bar" target tiles followed by the stashed item stacks,
 * wrapping onto extra rows. Right-clicking the background offers bar-wide
 * operations.
 */
export function TopSection({ items, showAddToGrid = true, onAddClick }: TopSectionProps) {
  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>
        <div
          data-drop-id="dropbar"
          style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
          className="flex max-h-[184px] flex-wrap items-start gap-1 overflow-y-auto px-3 py-2"
        >
          {showAddToGrid && (
            <DashedTile dropId="add-to-grid" label="Add to Grid" onClick={onAddClick}>
              <Plus className="size-6 text-neutral-300" strokeWidth={2} />
            </DashedTile>
          )}
          <DashedTile dropId="dropbar" label="Drop Bar">
            <ArrowDownToLine className="size-6 text-neutral-300" strokeWidth={2} />
          </DashedTile>
          {items.map((item) => (
            <DropBarTile
              key={item.id}
              item={item}
              onRemove={(id) => backend.dropBar.remove(id)}
            />
          ))}
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem
          disabled={items.filter((i) => i.kind === "files").length < 2}
          onClick={() => backend.dropBar.combineAll()}
        >
          Combine all Items to Stack
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          variant="destructive"
          disabled={items.length === 0}
          onClick={() => backend.dropBar.clear()}
        >
          Clear Drop Bar
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  )
}
