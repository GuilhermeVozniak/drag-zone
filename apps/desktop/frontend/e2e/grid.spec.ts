import { expect, test } from "@playwright/test";

// Drives the real rendered grid UI (GridPanel/AddTargetDialog/TaskList)
// against the stateful mock backend (e2e/mock/backend.ts) — no native OS
// file-drop is simulated here (that can't be driven from a browser), so
// "task progress" is triggered via a tile click instead: "Shorten URL"
// declares a "clicked" event, so clickBehavior() routes the click to
// backend.click() -> the mock's ClickTarget -> startTask(), which is the
// same event the mock's DropOnTarget path would produce.
//
// Note: the mock's `shares` state is only ever emptied (ClearRecentShares);
// no click-driven action sets a task's resultUrl or pushes to `shares`, so
// there is no genuine path here to exercise the Recently Shared pill. That
// coverage is deferred to Task 23 (whichever action wires up a real share
// result in the mock).

test("add a target from the catalogue, configure it, and see it in the grid", async ({ page }) => {
  await page.goto("/");

  await page.locator('[data-drop-id="add-to-grid"]').click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();

  await dialog.getByText("Shorten URL", { exact: true }).click();

  const nameInput = dialog.getByRole("textbox").first();
  await expect(nameInput).toHaveValue("Shorten URL");
  await nameInput.fill("Shorten Test");

  await dialog.getByRole("button", { name: "Add to Grid" }).click();
  await expect(dialog).toBeHidden();

  await expect(page.getByRole("button", { name: "Shorten Test" })).toBeVisible();
});

test("clicking a clickable tile runs a task that progresses to completion", async ({ page }) => {
  await page.goto("/");

  // Seed the target through the same UI path as the previous test.
  await page.locator('[data-drop-id="add-to-grid"]').click();
  const dialog = page.getByRole("dialog");
  await dialog.getByText("Shorten URL", { exact: true }).click();
  await dialog.getByRole("button", { name: "Add to Grid" }).click();
  await expect(dialog).toBeHidden();

  const tile = page.getByRole("button", { name: "Shorten URL" });
  await expect(tile).toBeVisible();

  // Click runs the action (mock's ClickTarget -> startTask): a TASK
  // PROGRESS row appears for it immediately at 0%...
  await tile.click();

  await expect(page.getByText("TASK PROGRESS")).toBeVisible();
  const taskRow = page.getByText("Shorten URL…");
  await expect(taskRow).toBeVisible();

  // ...and the mock auto-completes it ~50ms later: TaskList's cancel/dismiss
  // button flips from "Cancel" (while task.status === "running") to
  // "Dismiss" (once status === "done"), which is a more reliable completion
  // signal here than the progress bar itself — components/ui/progress.tsx
  // destructures `value` out of props but never forwards it to
  // ProgressPrimitive.Root, so the bar always renders as indeterminate
  // regardless of the real percent (pre-existing bug, out of scope here).
  const dismissButton = page.getByTitle("Dismiss");
  await expect(dismissButton).toBeVisible();

  // Dismissing the finished task clears the section entirely.
  await dismissButton.click();
  await expect(page.getByText("TASK PROGRESS")).toBeHidden();
});
