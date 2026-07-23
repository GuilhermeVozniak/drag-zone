import { afterEach, describe, expect, it } from "vitest";
import { gridInputBlocked, modalOpen, settingsViewOpen } from "@/lib/uistate";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("uistate", () => {
  it("reports no modal and no settings view on an empty document", () => {
    expect(modalOpen()).toBe(false);
    expect(settingsViewOpen()).toBe(false);
    expect(gridInputBlocked()).toBe(false);
  });

  it("detects an open dialog", () => {
    document.body.innerHTML = `<div role="dialog" data-state="open"></div>`;
    expect(modalOpen()).toBe(true);
    expect(gridInputBlocked()).toBe(true);
  });

  it("ignores a closed dialog", () => {
    document.body.innerHTML = `<div role="dialog" data-state="closed"></div>`;
    expect(modalOpen()).toBe(false);
    expect(gridInputBlocked()).toBe(false);
  });

  it("detects an open alertdialog", () => {
    document.body.innerHTML = `<div role="alertdialog" data-state="open"></div>`;
    expect(modalOpen()).toBe(true);
  });

  it("detects the settings view covering the grid", () => {
    document.body.innerHTML = `<div role="dialog" aria-label="Settings"></div>`;
    expect(settingsViewOpen()).toBe(true);
    expect(gridInputBlocked()).toBe(true);
  });
});
