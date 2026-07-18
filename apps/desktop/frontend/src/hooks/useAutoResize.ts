import { type RefObject, useEffect, useRef } from "react";
import { backend } from "@/lib/backend";

/**
 * Extra vertical space between the measured panel content and the native
 * window: PanelChrome's outer `pt-2` beak offset plus the panel's border.
 * Keep in sync with PanelChrome.tsx's layout.
 */
const CHROME_PX = 14;

/** Mirrors the clamp in the Go ResizeWindow binding (app_settings.go). */
export const AUTO_RESIZE_MIN_HEIGHT = 120;
export const AUTO_RESIZE_MAX_HEIGHT = 640;

/** The panel's own max-height, so overflow (and its inner scroll areas)
 * kicks in before the window would need to grow past AUTO_RESIZE_MAX_HEIGHT. */
export const PANEL_MAX_CONTENT_HEIGHT = AUTO_RESIZE_MAX_HEIGHT - CHROME_PX;

/**
 * Observes an element's natural (content-driven) height and resizes the
 * native window to fit it — Dropzone-style compact grid instead of a fixed
 * size with empty space. Guards against resize loops by only calling the
 * backend when the rounded target height actually changes.
 */
export function useAutoResize(ref: RefObject<HTMLElement | null>) {
  const lastHeight = useRef<number | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const measure = () => {
      // getBoundingClientRect() already reflects the CSS `zoom` scale used
      // for the grid-size setting; don't multiply by scale again.
      const contentHeight = el.getBoundingClientRect().height;
      const target = Math.min(
        AUTO_RESIZE_MAX_HEIGHT,
        Math.max(AUTO_RESIZE_MIN_HEIGHT, Math.ceil(contentHeight + CHROME_PX)),
      );
      if (target === lastHeight.current) return;
      lastHeight.current = target;
      backend.window.resize(target);
    };

    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, [ref]);
}
