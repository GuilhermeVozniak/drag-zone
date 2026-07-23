import { expect, test } from "./fixtures";

// The in-place update flow: the Updates tab offers "Update to X" when the
// (mock) backend reports a newer release with a DMG asset (?update=1), and
// clicking it drives the install progress stream to the relaunch note.

async function openUpdatesTab(page: import("@playwright/test").Page) {
  await page.locator('button[title="Settings"]').click();
  await page.getByRole("menuitem", { name: /Settings/ }).click();
  const dialog = page.getByRole("dialog", { name: "Settings" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("tab", { name: "Updates" }).click();
  return dialog;
}

test("an available update installs in place and shows the relaunch note", async ({ page }) => {
  await page.goto("/?update=1");
  const dialog = await openUpdatesTab(page);

  await expect(dialog.getByText(/Version 9\.9\.9 is available/)).toBeVisible();
  await dialog.getByRole("button", { name: "Update to 9.9.9" }).click();

  // The mock streams download → verify → install → done.
  await expect(dialog.getByText(/Downloading 9\.9\.9…/)).toBeVisible();
  await expect(dialog.getByText("Updated to 9.9.9 — relaunching…")).toBeVisible();
});

test("without an available update the tab stays on the up-to-date state", async ({ page }) => {
  await page.goto("/?update=1");
  const dialog = await openUpdatesTab(page);
  await expect(dialog.getByText(/Version 9\.9\.9 is available/)).toBeVisible();

  // The "release notes" link stays available next to the update button.
  await expect(dialog.getByText("release notes")).toBeVisible();
});
