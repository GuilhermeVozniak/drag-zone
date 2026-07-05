import { Plus, ArrowDownToLine } from "lucide-react"

/**
 * The empty-state of the top section, matching Dropzone 4: two dashed
 * rounded-square drop tiles — "Add to Grid" (drop folders/apps/.dzbundles,
 * click to open the catalogue) and "Drop Bar" (drop files to stash them).
 */
export function EmptyTopTiles({ onAddClick }: { onAddClick: () => void }) {
  return (
    <div className="flex gap-4 px-4 py-3">
      <button
        data-drop-id="add-to-grid"
        onClick={onAddClick}
        style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
        className="flex w-[76px] flex-col items-center gap-1.5"
      >
        <span className="flex size-[52px] items-center justify-center rounded-xl border-2 border-dashed border-neutral-500 transition-colors hover:border-neutral-300">
          <Plus className="size-6 text-neutral-300" strokeWidth={2} />
        </span>
        <span className="text-[11px] text-neutral-400">Add to Grid</span>
      </button>
      <div
        data-drop-id="dropbar"
        style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
        className="flex w-[76px] flex-col items-center gap-1.5"
      >
        <span className="flex size-[52px] items-center justify-center rounded-xl border-2 border-dashed border-neutral-500">
          <ArrowDownToLine className="size-6 text-neutral-300" strokeWidth={2} />
        </span>
        <span className="text-[11px] text-neutral-400">Drop Bar</span>
      </div>
    </div>
  )
}
