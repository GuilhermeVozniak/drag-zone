import type { DropBarItem } from "@/lib/backend"
import { cn } from "@/lib/utils"
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area"
import { DropBarTile } from "./DropBarTile"

interface DropBarProps {
  items: DropBarItem[]
  onRemove: (id: string) => void
  onClear: () => void
}

/** The shelf at the top of the grid holding stashed files, text, and URLs. */
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
