import { expect, test } from "./fixtures";

// Proves the whole e2e pipeline works end to end: `vite build --mode e2e`
// resolves wailsjs to the stateful mock (e2e/mock/backend.ts) instead of
// generated bindings, the built app boots against it, and the grid renders
// its persistent tiles from the mock's seeded state.
test("grid renders with the persistent Add to Grid and Drop Bar tiles", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("Add to Grid")).toBeVisible();
  await expect(page.getByText("Drop Bar")).toBeVisible();
});

test("seeded grid targets render", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("Downloads")).toBeVisible();
  await expect(page.getByText("Desktop")).toBeVisible();
});
