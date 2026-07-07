import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GeneralTab } from "@/features/settings/GeneralTab";
import type { Settings } from "@/lib/backend";

vi.mock("@/lib/backend");

const base = (): Settings =>
  ({
    gridSize: 33,
    gridColumns: 4,
    globalShortcut: "F3",
    popOutShortcut: "F4",
    theme: "system",
    animateGrid: true,
    showKeyOverlays: false,
    launchAtLogin: false,
    notifyOnComplete: true,
    playSounds: true,
    dragOverlay: true,
    dropBarKeepsItems: false,
  }) as Settings;

const switchInRow = (label: string) =>
  within(screen.getByText(label).closest("div") as HTMLElement).getByRole("switch");

beforeEach(() => vi.clearAllMocks());

describe("GeneralTab", () => {
  it("enables launch-at-login", async () => {
    const user = userEvent.setup();
    const update = vi.fn();
    render(<GeneralTab settings={base()} update={update} />);
    await user.click(switchInRow("Launch at login"));
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ launchAtLogin: true }));
  });

  it("switches to forced dark mode", async () => {
    const user = userEvent.setup();
    const update = vi.fn();
    render(<GeneralTab settings={base()} update={update} />);
    await user.click(switchInRow("Always use dark mode"));
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ theme: "dark" }));
  });

  it("turns off play-sounds", async () => {
    const user = userEvent.setup();
    const update = vi.fn();
    render(<GeneralTab settings={base()} update={update} />);
    await user.click(switchInRow("Play sounds"));
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ playSounds: false }));
  });

  it("enables keep-drop-bar-items", async () => {
    const user = userEvent.setup();
    const update = vi.fn();
    render(<GeneralTab settings={base()} update={update} />);
    await user.click(switchInRow("Keep Drop Bar items after drag out"));
    expect(update).toHaveBeenCalledWith(expect.objectContaining({ dropBarKeepsItems: true }));
  });
});
