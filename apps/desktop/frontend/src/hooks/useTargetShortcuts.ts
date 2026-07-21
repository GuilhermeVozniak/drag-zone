import { useEffect } from "react";
import { backend, type Target } from "@/lib/backend";
import { gridInputBlocked } from "@/lib/uistate";

/**
 * Launches targets by their single-key shortcut while the grid is open,
 * ignoring keystrokes aimed at inputs, held with command/control/option, or
 * landing while a modal dialog or the settings view covers the grid (the
 * grid stays mounted underneath both).
 */
export function useTargetShortcuts(targets: Target[]) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const el = e.target as HTMLElement;
      if (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable) return;
      if (gridInputBlocked()) return;
      const key = e.key.length === 1 ? e.key.toUpperCase() : "";
      if (!key) return;
      const match = targets.find((t) => t.shortcut?.toUpperCase() === key);
      if (match) {
        e.preventDefault();
        backend.click(match.id);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [targets]);
}
