import { expect, test } from "./fixtures";

// Task lifecycle scenarios: a running task the user cancels mid-flight, and
// a script-style input prompt that blocks the task until answered. Both use
// the mock backend's "(hang)"/"(prompt)" label hooks (see startTask in
// e2e/mock/backend.ts); the tile click path is what starts the task.

async function addShortenTarget(page: import("@playwright/test").Page, name: string) {
  await page.locator('[data-drop-id="add-to-grid"]').click();
  const dialog = page.getByRole("dialog");
  await dialog.getByText("Shorten URL", { exact: true }).click();
  const nameInput = dialog.getByRole("textbox").first();
  await nameInput.fill(name);
  await dialog.getByRole("button", { name: "Add to Grid" }).click();
  await expect(dialog).toBeHidden();
}

test("cancelling a running task shows a neutral cancelled row, then dismisses", async ({
  page,
}) => {
  await page.goto("/");
  await addShortenTarget(page, "Hang (hang)");

  await page.getByRole("button", { name: "Hang (hang)" }).click();
  await expect(page.getByText("TASK PROGRESS")).toBeVisible();
  await expect(page.getByText("Hang (hang)…")).toBeVisible();

  // The "(hang)" task never completes on its own; cancel it.
  await page.getByTitle("Cancel").click();

  // Cancelled reads as a neutral state — not a red error (the old backend
  // reported cancellations as "error: cancelled" with a failure sound).
  const row = page.getByText("Hang (hang) — cancelled");
  await expect(row).toBeVisible();
  await expect(row).not.toHaveClass(/text-red-400/);

  // It must stay cancelled — the mock's auto-complete must not revive it.
  await page.waitForTimeout(150);
  await expect(row).toBeVisible();

  // The action button is now Dismiss, and it clears the section.
  await page.getByTitle("Dismiss").click();
  await expect(page.getByText("TASK PROGRESS")).toBeHidden();
});

test("a script input prompt pauses the task until answered", async ({ page }) => {
  await page.goto("/");
  await addShortenTarget(page, "Ask (prompt)");

  await page.getByRole("button", { name: "Ask (prompt)" }).click();

  // The input:request event opens the prompt dialog; the task stays running.
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText("Enter a value")).toBeVisible();
  await expect(page.getByText("Ask (prompt)…")).toBeVisible();

  // Answering completes the task with the typed value as its detail.
  await dialog.getByRole("textbox").fill("hello world");
  await dialog.getByRole("button", { name: "OK" }).click();
  await expect(page.getByText("Ask (prompt) — hello world")).toBeVisible();
  await expect(page.getByTitle("Dismiss")).toBeVisible();
});

test("cancelling the prompt dialog leaves the task running", async ({ page }) => {
  await page.goto("/");
  await addShortenTarget(page, "Ask (prompt)");

  await page.getByRole("button", { name: "Ask (prompt)" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();

  // Dismissing the dialog answers not-OK; the mock completes the task
  // without a detail — the point is the dialog closes and the task leaves
  // the running state instead of hanging forever.
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByTitle("Dismiss")).toBeVisible();
});
