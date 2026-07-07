import { vi } from "vitest";

type FileDropCb = (x: number, y: number, paths: string[]) => void;
let fileDropCb: FileDropCb | null = null;

export const OnFileDrop = vi.fn((cb: FileDropCb, _useDropTarget: boolean) => {
  fileDropCb = cb;
});
export const EventsOn = vi.fn((_event: string, _cb: (...a: unknown[]) => void) => () => {});
export const EventsEmit = vi.fn();

// --- test helpers (not part of the real runtime API) ---
export function __emitFileDrop(x: number, y: number, paths: string[]) {
  fileDropCb?.(x, y, paths);
}
export function __resetRuntimeStub() {
  fileDropCb = null;
  OnFileDrop.mockClear();
  EventsOn.mockClear();
  EventsEmit.mockClear();
}
