import { ArrowDown } from "lucide-react";
import { cn } from "@/lib/utils";

interface DropTargetOverlayProps {
  /** Whether a file drag is currently over the grid (native drag:active
   * signal, gated by the "Show drag target overlay" setting upstream). */
  active: boolean;
}

/**
 * Dropzone-4-style "drop here" affordance shown over the grid while a
 * native (Finder) file drag is in progress over it. Always mounted so it
 * can animate in/out on the `active` flag instead of popping; it is
 * `pointer-events-none` throughout so it never intercepts the actual drop —
 * that is still resolved by useNativeFileDrop via elementsFromPoint. The
 * background is translucent so the tiles underneath stay visible: users
 * still aim at a specific Drop Bar tile to stack onto it.
 */
export function DropTargetOverlay({ active }: DropTargetOverlayProps) {
  return (
    <div
      aria-hidden="true"
      data-state={active ? "active" : "inactive"}
      className={cn(
        "pointer-events-none absolute inset-0 z-20 m-2 flex flex-col items-center",
        "justify-center gap-2 rounded-xl border-2 border-dashed",
        "transition-opacity duration-150 ease-out",
        active ? "opacity-100" : "opacity-0",
      )}
      style={{
        background: "color-mix(in srgb, var(--panel-bg) 45%, transparent)",
        borderColor: "var(--drop-overlay-border)",
      }}
    >
      <ArrowDown className="size-8 text-neutral-300" />
      <p className="text-sm font-medium text-neutral-200">Drop to add</p>
    </div>
  );
}
