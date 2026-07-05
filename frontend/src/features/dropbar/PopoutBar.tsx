import { backend } from "@/lib/backend"
import { useDropBar } from "@/hooks/useBackend"
import { PanelTopClose } from "lucide-react"
import { DropBar } from "./DropBar"

/** Compact always-on-top Drop Bar shown when popped out of the grid. */
export function PopoutBar() {
  const items = useDropBar()
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header
        className="flex items-center justify-between px-4 py-2"
        style={{ "--wails-draggable": "drag" } as React.CSSProperties}
      >
        <span className="text-xs font-semibold tracking-wide text-neutral-400">
          Drop Bar
        </span>
        <button
          onClick={() => backend.dropBar.setPopOut(false)}
          className="flex size-6 items-center justify-center rounded-full hover:bg-white/10"
          title="Dock back into grid"
        >
          <PanelTopClose className="size-3.5 text-neutral-400" />
        </button>
      </header>
      <DropBar
        items={items}
        onRemove={(id) => backend.dropBar.remove(id)}
        onClear={() => backend.dropBar.clear()}
      />
    </div>
  )
}
