import { test as base } from "@playwright/test";

// Shared fixtures for all e2e specs. Import `test`/`expect` from here, not
// from @playwright/test.
//
// The init script disables CSS animations/transitions in every page:
// headless Chromium throttles animation events on elements occluded by an
// overlay (e.g. a dropdown menu closing as the settings view mounts over
// it), so Radix Presence sometimes never receives `animationend` and the
// "closed" menu stays mounted — the next trigger click then toggles a menu
// Radix still considers open, and it instantly dismisses. Real (headed)
// browsers are unaffected. As a side benefit, specs never wait on
// animation timing.
export const test = base.extend({
  context: async ({ context }, use) => {
    await context.addInitScript(() => {
      const style = document.createElement("style");
      style.textContent =
        "*, *::before, *::after { animation: none !important; transition: none !important; }";
      if (document.head) {
        document.head.appendChild(style);
      } else {
        document.addEventListener("DOMContentLoaded", () => document.head.appendChild(style));
      }
    });
    await use(context);
  },
});

export { expect } from "@playwright/test";
