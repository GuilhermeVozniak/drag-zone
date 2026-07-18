import { type ReactNode, useEffect, useRef, useState } from "react";
import { PANEL_MAX_CONTENT_HEIGHT, useAutoResize } from "@/hooks/useAutoResize";
import { useSettings } from "@/hooks/useBackend";
import { events } from "@/lib/backend";
import { cn } from "@/lib/utils";

/**
 * The popover-style window chrome: a beak/arrow pointing at the menu bar
 * icon (its x position is reported by the native layer, since the window
 * clamps at screen edges) and a slide-down entrance replayed every time the
 * grid is shown.
 *
 * The panel below sizes to its natural content height (no `h-screen`/`flex-1`
 * stretching) so it can be measured and used to resize the native window to
 * fit — see useAutoResize. It's capped at PANEL_MAX_CONTENT_HEIGHT so very
 * tall content scrolls internally instead of growing the window unbounded.
 */
export function PanelChrome({ children }: { children: ReactNode }) {
  const [beakX, setBeakX] = useState<number | null>(null);
  const [showKey, setShowKey] = useState(0);
  const [settings] = useSettings();
  const animate = settings?.animateGrid ?? true;
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const offBeak = events.onWindowBeak(setBeakX);
    const offVis = events.onWindowVisibility((visible) => {
      if (visible) setShowKey((k) => k + 1);
    });
    return () => {
      offBeak();
      offVis();
    };
  }, []);

  useAutoResize(panelRef);

  return (
    <div className="relative flex flex-col pt-2">
      <div className="absolute top-0 z-10 -translate-x-1/2" style={{ left: beakX ?? "50%" }}>
        <div
          className="h-0 w-0 border-x-[9px] border-b-[9px] border-x-transparent"
          style={{ borderBottomColor: "var(--panel-bg)" }}
        />
      </div>
      <div
        ref={panelRef}
        key={animate ? showKey : 0}
        className={cn(
          "flex w-full flex-col overflow-hidden rounded-lg border border-white/10 shadow-2xl",
          animate && "animate-in fade-in slide-in-from-top-2 duration-200",
        )}
        style={{ background: "var(--panel-bg)", maxHeight: PANEL_MAX_CONTENT_HEIGHT }}
      >
        {children}
      </div>
    </div>
  );
}
