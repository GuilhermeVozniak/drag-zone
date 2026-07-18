import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PopoutBar } from "@/features/dropbar/PopoutBar";
import { backend, type DropBarItem } from "@/lib/backend";

vi.mock("@/lib/backend");

beforeEach(() => vi.clearAllMocks());

describe("PopoutBar", () => {
  it("renders the compact pinned header and its drop target, without the Add to Grid tile", async () => {
    vi.mocked(backend.dropBar.list).mockResolvedValue([] as never);
    render(<PopoutBar />);
    expect(screen.getAllByText("Drop Bar").length).toBeGreaterThan(0);
    expect(screen.queryByText("Add to Grid")).not.toBeInTheDocument();
    const dropTarget = document.querySelector("[data-drop-id='dropbar']");
    expect(dropTarget).not.toBeNull();
    expect((dropTarget as HTMLElement).style.getPropertyValue("--wails-drop-target")).toBe("drop");
    await waitFor(() => expect(backend.dropBar.list).toHaveBeenCalled());
  });

  it("renders the stashed items pulled from the backend", async () => {
    vi.mocked(backend.dropBar.list).mockResolvedValue([
      {
        id: "p1",
        kind: "files",
        paths: ["/unique/popoutbar/p1.txt"],
        label: "pinned.txt",
        locked: false,
      } as DropBarItem,
    ] as never);
    render(<PopoutBar />);
    await waitFor(() => expect(screen.getByText("pinned.txt")).toBeInTheDocument());
  });

  it("docks back into the grid when the close button is clicked", async () => {
    vi.mocked(backend.dropBar.list).mockResolvedValue([] as never);
    const user = userEvent.setup();
    render(<PopoutBar />);
    await user.click(screen.getByTitle("Dock back into grid"));
    expect(backend.dropBar.setPopOut).toHaveBeenCalledWith(false);
  });
});
