import { useEffect, useRef } from "react";
import { backend } from "@/lib/backend";

/**
 * Extra vertical space between the measured panel content and the native
 * window: PanelChrome's outer `pt-2` beak offset. The panel's border is
 * already included in the measured rect (getBoundingClientRect covers the
 * border box). Keep in sync with PanelChrome.tsx's layout.
 */
const CHROME_PX = 8;

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
 * backend when the rounded target height actually changes. Disabled in
 * settings mode, where the settings window owns its size (the Go
 * ResizeWindow binding also ignores calls then).
 *
 * Takes the element itself (via a callback ref / state) rather than a
 * RefObject: PanelChrome remounts the panel div on every show to replay the
 * entrance animation, and the observer must follow the new element — a
 * stable RefObject would keep observing the detached node, whose collapse
 * to 0px would shrink the window to the minimum height.
 */
export function useAutoResize(el: HTMLElement | null, enabled = true) {
  const lastHeight = useRef<number | null>(null);

  useEffect(() => {
    if (!enabled || !el) return;

    const measure = () => {
      // Skip stale notifications from a node that was just unmounted (its
      // rect collapses to 0) — measuring it would clamp the window to the
      // minimum height right as the panel reappears.
      if (!el.isConnected) return;
      // getBoundingClientRect() already reflects the CSS `zoom` scale used
      // for the grid-size setting; don't multiply by scale again.
      const contentHeight = el.getBoundingClientRect().height;
      if (contentHeight === 0) return;
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
  }, [el, enabled]);
}
