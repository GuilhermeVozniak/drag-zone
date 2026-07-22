import { expect, test } from "./fixtures";

// Target management via the tile context menu: Edit…, Duplicate, and
// Remove from Grid. The seeded grid starts with Downloads/Desktop/Zip/
// Trash/AirDrop (see e2e/mock/backend.ts).

test("duplicating a tile adds an independent copy", async ({ page }) => {
  await page.goto("/");

  const tile = page.getByRole("button", { name: "Downloads" });
  await tile.click({ button: "right" });
  await page.getByRole("menuitem", { name: "Duplicate" }).click();

  await expect(page.getByRole("button", { name: "Downloads" })).toHaveCount(2);
});

test("editing a tile renames it and changes its folder", async ({ page }) => {
  await page.goto("/");

  const tile = page.getByRole("button", { name: "Downloads" });
  await tile.click({ button: "right" });
  await page.getByRole("menuitem", { name: "Edit…" }).click();

  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();

  const nameInput = dialog.getByRole("textbox").first();
  await expect(nameInput).toHaveValue("Downloads");
  await nameInput.fill("My Stuff");
  await dialog.getByRole("button", { name: "Save" }).click();

  await expect(dialog).toBeHidden();
  await expect(page.getByRole("button", { name: "My Stuff" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Downloads" })).toBeHidden();
});

test("removing a tile takes it out of the grid", async ({ page }) => {
  await page.goto("/");

  const tile = page.getByRole("button", { name: "Zip Files" });
  await expect(tile).toBeVisible();
  await tile.click({ button: "right" });
  await page.getByRole("menuitem", { name: "Remove from Grid" }).click();

  await expect(page.getByRole("button", { name: "Zip Files" })).toBeHidden();
});

test("the seeded grid shows all five default targets", async ({ page }) => {
  await page.goto("/");

  for (const name of ["Downloads", "Desktop", "Zip Files", "Move to Trash", "AirDrop"]) {
    await expect(page.getByRole("button", { name, exact: true })).toBeVisible();
  }
});
