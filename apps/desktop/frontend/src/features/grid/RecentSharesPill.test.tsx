import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { backend, type Share } from "@/lib/backend";
import { RecentSharesPill } from "./RecentSharesPill";

vi.mock("@/lib/backend");

const share = (over: Partial<Share> = {}): Share =>
  ({ title: "Share", url: "https://x.test", at: null, ...over }) as Share;

beforeEach(() => vi.clearAllMocks());

describe("RecentSharesPill", () => {
  it("renders nothing while there are no recent shares", async () => {
    vi.mocked(backend.shares.list).mockResolvedValue([] as never);
    const { container } = render(<RecentSharesPill />);
    // Let the initial backend.shares.list() promise resolve.
    await vi.waitFor(() => expect(backend.shares.list).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("renders shares newest-first and opens a share's URL via the backend on click", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.shares.list).mockResolvedValue([
      share({ title: "Newest", url: "https://newest.test/a" }),
      share({ title: "Older", url: "https://older.test/b" }),
    ] as never);
    render(<RecentSharesPill />);

    const trigger = await screen.findByRole("button", { name: /recently shared/i });
    await user.click(trigger);

    const items = await screen.findAllByRole("menuitem");
    const shareItems = items.filter((i) => i.textContent?.includes("https://"));
    expect(shareItems).toHaveLength(2);
    expect(shareItems[0].textContent).toContain("https://newest.test/a");
    expect(shareItems[1].textContent).toContain("https://older.test/b");

    await user.click(shareItems[0]);
    expect(backend.shares.open).toHaveBeenCalledWith("https://newest.test/a");
    expect(backend.shares.open).not.toHaveBeenCalledWith("https://older.test/b");
  });

  it("clears the menu via the backend when Clear Menu is clicked", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.shares.list).mockResolvedValue([share()] as never);
    render(<RecentSharesPill />);

    const trigger = await screen.findByRole("button", { name: /recently shared/i });
    await user.click(trigger);
    await user.click(await screen.findByText("Clear Menu"));
    expect(backend.shares.clear).toHaveBeenCalledTimes(1);
  });
});
