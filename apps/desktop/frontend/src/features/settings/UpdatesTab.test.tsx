import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UpdatesTab } from "@/features/settings/UpdatesTab";
import { backend, type Settings, type UpdateInfo } from "@/lib/backend";

vi.mock("@/lib/backend");

const settings = { autoUpdateCheck: true } as Settings;
const info = (over: Partial<UpdateInfo>): UpdateInfo =>
  ({ available: false, latest: "", current: "", url: "", downloadUrl: "", ...over }) as UpdateInfo;

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(backend.updates.version).mockResolvedValue("v0.3.8" as never);
});

describe("UpdatesTab", () => {
  it("checks on mount and offers the download when an update is available", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.updates.check).mockResolvedValue(
      info({
        available: true,
        latest: "v0.4.0",
        url: "https://gh/notes",
        downloadUrl: "https://gh/dl",
      }) as never,
    );
    render(<UpdatesTab settings={settings} update={vi.fn()} />);
    await waitFor(() => expect(screen.getByText(/v0\.4\.0 is available/)).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /Download v0\.4\.0/ }));
    expect(backend.openURL).toHaveBeenCalledWith("https://gh/dl");
  });

  it("reports up-to-date when no newer version exists", async () => {
    vi.mocked(backend.updates.check).mockResolvedValue(info({ available: false }) as never);
    render(<UpdatesTab settings={settings} update={vi.fn()} />);
    await waitFor(() => expect(screen.getByText(/up to date/i)).toBeInTheDocument());
  });

  it("surfaces a check error", async () => {
    vi.mocked(backend.updates.check).mockRejectedValue(new Error("offline") as never);
    render(<UpdatesTab settings={settings} update={vi.fn()} />);
    await waitFor(() => expect(screen.getByText(/offline/)).toBeInTheDocument());
  });

  it("re-checks when Check Now is pressed", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.updates.check).mockResolvedValue(info({ available: false }) as never);
    render(<UpdatesTab settings={settings} update={vi.fn()} />);
    await waitFor(() => expect(backend.updates.check).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole("button", { name: /Check Now/ }));
    await waitFor(() => expect(backend.updates.check).toHaveBeenCalledTimes(2));
  });
});
