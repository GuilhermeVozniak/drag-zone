import { expect, test } from "./fixtures";

// Drop Bar operations: Cmd-V paste, stack separate/combine, rename, and
// lock. The seeded Drop Bar starts with one "Shared link" URL item (see
// e2e/mock/backend.ts); the mock's DropBarPaste stashes a two-file stack.

test("Cmd-V pastes the clipboard as a file stack", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByText("Shared link", { exact: true })).toBeVisible();
  await page.keyboard.press("Meta+v");

  // The mock clipboard holds two files: they land as one "2 Items" stack
  // with a count badge.
  await expect(page.getByText("2 Items", { exact: true })).toBeVisible();
});

test("a stack separates into singles and combines back", async ({ page }) => {
  await page.goto("/");
  await page.keyboard.press("Meta+v");
  await expect(page.getByText("2 Items", { exact: true })).toBeVisible();

  // Separate via the tile's context menu: two single-file items appear.
  await page.getByText("2 Items", { exact: true }).click({ button: "right" });
  await page.getByRole("menuitem", { name: "Separate Items" }).click();
  await expect(page.getByText("report.pdf", { exact: true })).toBeVisible();
  await expect(page.getByText("notes.txt", { exact: true })).toBeVisible();
  await expect(page.getByText("2 Items", { exact: true })).toBeHidden();

  // Combine them back via the bar's context menu.
  await page
    .locator('div[data-drop-id="dropbar"]')
    .click({ button: "right", position: { x: 5, y: 5 } });
  await page.getByRole("menuitem", { name: "Combine all Items to Stack" }).click();
  await expect(page.getByText("2 Items", { exact: true })).toBeVisible();
  await expect(page.getByText("report.pdf", { exact: true })).toBeHidden();
});

test("renaming a stack updates its label", async ({ page }) => {
  await page.goto("/");
  await page.keyboard.press("Meta+v");
  await expect(page.getByText("2 Items", { exact: true })).toBeVisible();

  await page.getByText("2 Items", { exact: true }).click({ button: "right" });
  await page.getByRole("menuitem", { name: "Name Stack…" }).click();

  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  const input = dialog.getByRole("textbox");
  await input.fill("Project Files");
  await dialog.getByRole("button", { name: "Save" }).click();

  await expect(dialog).toBeHidden();
  await expect(page.getByText("Project Files", { exact: true })).toBeVisible();
  await expect(page.getByText("2 Items", { exact: true })).toBeHidden();
});

test("locking an item shows the lock badge and flips the menu item", async ({ page }) => {
  await page.goto("/");

  const item = page.getByText("Shared link", { exact: true });
  await item.click({ button: "right" });
  await page.getByRole("menuitem", { name: "Lock Items" }).click();

  // Locked tiles show a small amber padlock badge.
  await expect(page.locator("svg.text-amber-400")).toBeVisible();

  // The context menu now offers to unlock instead.
  await item.click({ button: "right" });
  await expect(page.getByRole("menuitem", { name: "Unlock Items" })).toBeVisible();
});

test("the per-item X button removes just that item", async ({ page }) => {
  await page.goto("/");
  await page.keyboard.press("Meta+v");
  await expect(page.getByText("2 Items", { exact: true })).toBeVisible();

  // The X is revealed on hover (group-hover).
  const tile = page.getByText("2 Items", { exact: true }).locator("..");
  await tile.hover();
  await tile.locator("button").first().click();

  await expect(page.getByText("2 Items", { exact: true })).toBeHidden();
  await expect(page.getByText("Shared link", { exact: true })).toBeVisible();
});
