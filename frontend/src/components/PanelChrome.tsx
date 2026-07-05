import { useEffect, useState, type ReactNode } from "react"
import { events } from "@/lib/backend"

/**
 * The popover-style window chrome: a beak/arrow pointing at the menu bar
 * icon (its x position is reported by the native layer, since the window
 * clamps at screen edges) and a slide-down entrance replayed every time the
 * grid is shown.
 */
export function PanelChrome({ children }: { children: ReactNode }) {
  const [beakX, setBeakX] = useState<number | null>(null)
  const [showKey, setShowKey] = useState(0)

  useEffect(() => {
    const offBeak = events.onWindowBeak(setBeakX)
    const offVis = events.onWindowVisibility((visible) => {
      if (visible) setShowKey((k) => k + 1)
    })
    return () => {
      offBeak()
      offVis()
    }
  }, [])

  return (
    <div className="relative flex h-screen flex-col pt-2">
      <div
        className="absolute top-0 z-10 -translate-x-1/2"
        style={{ left: beakX ?? "50%" }}
      >
        <div className="h-0 w-0 border-x-[9px] border-b-[9px] border-x-transparent border-b-neutral-900" />
      </div>
      <div
        key={showKey}
        className="animate-in fade-in slide-in-from-top-2 flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-white/10 bg-neutral-900 shadow-2xl duration-200"
      >
        {children}
      </div>
    </div>
  )
}
