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
      className={cn("min-h-[84px] px-2 py-1.5")}
    >
      <ScrollArea className="w-full">
        <div className="flex items-start gap-1">
          {items.map((item) => (
            <DropBarTile key={item.id} item={item} onRemove={onRemove} />
          ))}
        </div>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
      <div className="flex justify-end">
        <button
          onClick={onClear}
          className="text-[10px] text-neutral-500 hover:text-neutral-300"
        >
          Clear all
        </button>
      </div>
    </div>
  )
}
