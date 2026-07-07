import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Radix UI primitives (Slider/Select/Dialog/Switch) call these at render or
// interaction time; jsdom implements none of them.
if (!globalThis.ResizeObserver) {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}
Element.prototype.scrollIntoView ||= () => {};
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.setPointerCapture ||= () => {};
Element.prototype.releasePointerCapture ||= () => {};

// jsdom doesn't implement elementFromPoint at all; provide a spy-able stub so
// hit-testing tests can override it via vi.spyOn(document, 'elementFromPoint').
if (!document.elementFromPoint) {
  document.elementFromPoint = () => null;
}

afterEach(() => {
  cleanup();
});
