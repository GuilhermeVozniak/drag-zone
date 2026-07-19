import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsView } from "@/features/settings/SettingsView";
import { type AddonInfo, backend, type Settings } from "@/lib/backend";

vi.mock("@/lib/backend");

const baseSettings = (): Settings =>
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
    autoUpdateCheck: true,
  }) as Settings;

const switchInRow = (label: string) =>
  within(screen.getByText(label).closest("div") as HTMLElement).getByRole("switch");

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(backend.settings.get).mockResolvedValue(baseSettings() as never);
  vi.mocked(backend.settings.set).mockResolvedValue(undefined as never);
  vi.mocked(backend.addons.list).mockResolvedValue([
    { name: "Zip It", installed: false } as AddonInfo,
  ] as never);
  vi.mocked(backend.cli.installed).mockResolvedValue(true as never);
});

describe("SettingsView", () => {
  it("shows the General tab by default", async () => {
    render(<SettingsView />);
    await waitFor(() => expect(screen.getByText("Grid size")).toBeInTheDocument());
    expect(screen.queryByText("Zip It")).not.toBeInTheDocument();
  });

  it("opens on the requested tab", async () => {
    render(<SettingsView tab="addons" />);
    await waitFor(() => expect(screen.getByText("Zip It")).toBeInTheDocument());
    expect(screen.queryByText("Grid size")).not.toBeInTheDocument();
  });

  it("switches to the Add-ons tab and shows the catalogue", async () => {
    const user = userEvent.setup();
    render(<SettingsView />);
    await waitFor(() => expect(screen.getByText("Grid size")).toBeInTheDocument());

    await user.click(screen.getByRole("tab", { name: "Add-on Actions" }));
    await waitFor(() => expect(screen.getByText("Zip It")).toBeInTheDocument());
    expect(screen.queryByText("Grid size")).not.toBeInTheDocument();
  });

  it("switches to the Command Line tab and shows CLI status", async () => {
    const user = userEvent.setup();
    render(<SettingsView />);
    await waitFor(() => expect(screen.getByText("Grid size")).toBeInTheDocument());

    await user.click(screen.getByRole("tab", { name: "Command Line" }));
    await waitFor(() =>
      expect(screen.getByText("Installed at /usr/local/bin/dz")).toBeInTheDocument(),
    );
    expect(screen.queryByText("Grid size")).not.toBeInTheDocument();
  });

  it("persists a settings change through the backend", async () => {
    const user = userEvent.setup();
    render(<SettingsView />);
    await waitFor(() => expect(screen.getByText("Grid size")).toBeInTheDocument());

    await user.click(switchInRow("Always use dark mode"));
    await waitFor(() =>
      expect(backend.settings.set).toHaveBeenCalledWith(expect.objectContaining({ theme: "dark" })),
    );
  });

  it("closes via the header button through the backend", async () => {
    const user = userEvent.setup();
    render(<SettingsView />);
    await waitFor(() => expect(screen.getByText("Grid size")).toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "Close Settings" }));
    expect(backend.settings.close).toHaveBeenCalled();
  });
});
