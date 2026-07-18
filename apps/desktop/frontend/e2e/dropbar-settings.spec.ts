import { expect, test } from "@playwright/test";

// Drives the Drop Bar, Settings dialog, and first-run Onboarding carousel
// against the stateful mock backend (e2e/mock/backend.ts) — the same mock
// grid.spec.ts and smoke.spec.ts use, seeded with one Drop Bar item ("Shared
// link", see backend.ts's `dropBarItems`) and `onboardingSeen: true` so the
// grid — not the carousel — is what every other spec sees.
//
// Two minimal, realistic mock additions were made for this spec (see
// e2e/mock/backend.ts):
//   - `?onboarding=1` seeds `onboardingSeen: false` at module init, giving
//     this file a clean way to reach the carousel without flipping the
//     default every other spec relies on.
//   - `startTask()` now mirrors the real backend (internal/tasks/runner.go's
//     OnResultURL -> app.go's addRecentShare): completing a shorten-url task
//     sets the task's resultUrl and pushes a Recently Shared entry, closing
//     the gap Task 22 deferred (there was previously no UI-drivable path to
//     a share).

test.describe("Drop Bar", () => {
  test("the seeded item renders", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("Shared link", { exact: true })).toBeVisible();
  });

  test("renaming an item via its context menu updates the label", async ({ page }) => {
    await page.goto("/");

    const tile = page.getByText("Shared link", { exact: true });
    await expect(tile).toBeVisible();
    await tile.click({ button: "right" });

    const renameItem = page.getByRole("menuitem", { name: "Rename…" });
    await expect(renameItem).toBeVisible();
    await renameItem.click();

    const dialog = page.getByRole("dialog", { name: "Rename Item" });
    await expect(dialog).toBeVisible();
    const input = dialog.getByRole("textbox");
    await expect(input).toHaveValue("Shared link");
    await input.fill("Renamed Link");
    await dialog.getByRole("button", { name: "Save" }).click();

    await expect(dialog).toBeHidden();
    await expect(page.getByText("Renamed Link", { exact: true })).toBeVisible();
    await expect(page.getByText("Shared link", { exact: true })).toBeHidden();
  });

  test("clearing via the background context menu removes all items", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("Shared link", { exact: true })).toBeVisible();

    // Right-click the Drop Bar's own container (not a tile) for the bar-wide
    // menu; `div[data-drop-id="dropbar"]` picks the container over the
    // dashed "Drop Bar" target tile, which shares the same drop-id but is a
    // <button> (see TopSection.tsx). Aiming at its top-left padding corner
    // (px-3 py-2, before any tile starts) keeps the right-click off every
    // tile, each of which has its own NESTED context menu that would
    // otherwise swallow the event before it reaches this bar-wide one.
    await page
      .locator('div[data-drop-id="dropbar"]')
      .click({ button: "right", position: { x: 5, y: 5 } });

    const clearItem = page.getByRole("menuitem", { name: "Clear Drop Bar" });
    await expect(clearItem).toBeVisible();
    await clearItem.click();

    await expect(page.getByText("Shared link", { exact: true })).toBeHidden();
  });
});

test.describe("Settings", () => {
  async function openSettings(page: import("@playwright/test").Page) {
    await page.locator('button[title="Settings"]').click();
    await page.getByRole("menuitem", { name: /Settings/ }).click();
    const dialog = page.getByRole("dialog", { name: "Settings" });
    await expect(dialog).toBeVisible();
    return dialog;
  }

  test("opens via the gear menu and every tab renders its content", async ({ page }) => {
    await page.goto("/");
    const dialog = await openSettings(page);

    // General is the default tab.
    await expect(dialog.getByText("Grid size")).toBeVisible();
    await expect(dialog.getByText("Launch at login")).toBeVisible();

    await dialog.getByRole("tab", { name: "Add-ons" }).click();
    await expect(
      dialog.getByText("Community actions from aptonic/dropzone4-actions.", { exact: false }),
    ).toBeVisible();
    await expect(dialog.getByText("google-drive")).toBeVisible();

    await dialog.getByRole("tab", { name: "Command Line" }).click();
    await expect(dialog.getByText("Not installed")).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Install Command Line Tool" })).toBeVisible();

    await dialog.getByRole("tab", { name: "Updates" }).click();
    await expect(dialog.getByText("You're up to date.")).toBeVisible();
    await expect(dialog.getByText("0.1.0-e2e")).toBeVisible();

    await dialog.getByRole("tab", { name: "General" }).click();
    await expect(dialog.getByText("Grid size")).toBeVisible();
  });

  test("toggling dark mode persists across closing and reopening the dialog", async ({ page }) => {
    await page.goto("/");
    const dialog = await openSettings(page);

    const darkModeSwitch = dialog.locator('label:has-text("Always use dark mode") + button');
    await expect(darkModeSwitch).toHaveAttribute("data-state", "unchecked");

    await darkModeSwitch.click();
    await expect(darkModeSwitch).toHaveAttribute("data-state", "checked");
    // The whole panel is themed via a class on <html> (see App.tsx), which
    // is the actual effect of the setting the switch controls.
    await expect(page.locator("html")).toHaveClass(/dark/);

    // Close (Escape) and reopen: SettingsDialog remounts nothing (it's kept
    // alive, just hidden), but reopening re-reads the same persisted
    // settings store either way — proving the change stuck via setSettings.
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();

    const reopened = await openSettings(page);
    const reopenedSwitch = reopened.locator('label:has-text("Always use dark mode") + button');
    await expect(reopenedSwitch).toHaveAttribute("data-state", "checked");
  });

  test("moving the grid-size slider updates its value", async ({ page }) => {
    await page.goto("/");
    const dialog = await openSettings(page);

    const slider = dialog.getByRole("slider");
    await expect(slider).toHaveAttribute("aria-valuenow", "33");

    await slider.focus();
    for (let i = 0; i < 10; i++) await page.keyboard.press("ArrowRight");

    await expect(slider).toHaveAttribute("aria-valuenow", "43");
  });
});

test.describe("Onboarding", () => {
  // `?onboarding=1` is a minimal, test-only mock hook (see e2e/mock/backend.ts)
  // that seeds `onboardingSeen: false` for this navigation only, so the
  // carousel gate in src/App.tsx (`showOnboarding`) actually trips here
  // without touching the default every other spec depends on.
  test("shows on first run, navigates with next/back, and dismiss hides it", async ({ page }) => {
    await page.goto("/?onboarding=1");

    await expect(page.getByText("Welcome to DragZone")).toBeVisible();
    await expect(page.getByText("Add to Grid")).toBeHidden();

    await page.getByRole("button", { name: "Next" }).click();
    await expect(page.getByText("Drop files onto actions")).toBeVisible();

    await page.getByRole("button", { name: "Back" }).click();
    await expect(page.getByText("Welcome to DragZone")).toBeVisible();

    // Walk to the final slide, where "Next" becomes "Get Started".
    for (let i = 0; i < 4; i++) await page.getByRole("button", { name: "Next" }).click();
    await expect(page.getByText("Add-ons & the dz CLI")).toBeVisible();

    const getStarted = page.getByRole("button", { name: "Get Started" });
    await expect(getStarted).toBeVisible();
    await getStarted.click();

    // Dismissing marks onboardingSeen (via setSettings) and the carousel is
    // replaced by the real grid in the same render.
    await expect(page.getByText("Welcome to DragZone")).toBeHidden();
    await expect(page.getByText("Add to Grid")).toBeVisible();
  });

  test("Skip dismisses the carousel immediately", async ({ page }) => {
    await page.goto("/?onboarding=1");

    await expect(page.getByText("Welcome to DragZone")).toBeVisible();
    await page.getByRole("button", { name: "Skip" }).click();

    await expect(page.getByText("Welcome to DragZone")).toBeHidden();
    await expect(page.getByText("Add to Grid")).toBeVisible();
  });
});

test.describe("Recently Shared (closes Task 22's deferral)", () => {
  test("running Shorten URL from a click surfaces the Recently Shared pill", async ({ page }) => {
    await page.goto("/");

    // Same seeding path as grid.spec.ts: add "Shorten URL" from the
    // catalogue, then click it (clickBehavior routes clickable actions to
    // backend.click() -> the mock's ClickTarget -> startTask()).
    await page.locator('[data-drop-id="add-to-grid"]').click();
    const addDialog = page.getByRole("dialog");
    await addDialog.getByText("Shorten URL", { exact: true }).click();
    await addDialog.getByRole("button", { name: "Add to Grid" }).click();
    await expect(addDialog).toBeHidden();

    await expect(page.getByText("Recently Shared")).toBeHidden();

    await page.getByRole("button", { name: "Shorten URL" }).click();
    await expect(page.getByText("TASK PROGRESS")).toBeVisible();

    const pill = page.getByRole("button", { name: /Recently Shared/ });
    await expect(pill).toBeVisible();

    await pill.click();
    await expect(page.getByRole("menuitem", { name: /tinyurl\.com\/e2e-demo/ })).toBeVisible();
  });
});
