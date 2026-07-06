import { expect, test } from "@playwright/test";

test("hero renders with the macOS download", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toContainText("drop");

  const dl = page.getByTestId("primary-download");
  await expect(dl).toHaveAttribute("data-platform", "darwin");
  await expect(dl).toHaveAttribute(
    "href",
    /\/releases\/download\/v\d+\.\d+\.\d+\/dragzone_\d+\.\d+\.\d+_darwin_universal\.dmg$/,
  );
});

test("shows the built-in actions and menu-bar features", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: /every built-in action/i })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Built for the menu bar" })).toBeVisible();
  for (const action of ["AirDrop", "Zip Files", "Upload anywhere"]) {
    await expect(page.getByRole("heading", { name: action })).toBeVisible();
  }
});

test("links to the GitHub source", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Source on GitHub" })).toHaveAttribute(
    "href",
    "https://github.com/GuilhermeVozniak/drag-zone",
  );
});
