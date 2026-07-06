import type { ActionSpec } from "@/lib/backend"

/** What clicking a grid tile should do, given its action's spec. */
export type ClickBehavior = "run" | "config" | "none"

/**
 * Dropzone runs an action's click handler on click. Actions that declare no
 * "clicked" event instead open their config dialog, and drag-only actions with
 * nothing to configure do nothing — rather than surfacing a "does not support
 * clicks" error. A missing spec defers to the backend ("run").
 */
export function clickBehavior(spec: ActionSpec | undefined): ClickBehavior {
  if (spec && !(spec.events ?? []).includes("clicked")) {
    return spec.options && spec.options.length > 0 ? "config" : "none"
  }
  return "run"
}
