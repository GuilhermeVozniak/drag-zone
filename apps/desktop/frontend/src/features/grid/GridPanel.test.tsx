import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GridPanel } from "@/features/grid/GridPanel";
import { type ActionSpec, backend, type Target } from "@/lib/backend";

vi.mock("@/lib/backend");

const target = (over: Partial<Target> = {}): Target =>
  ({
    id: "t1",
    actionId: "zip",
    label: "Tile",
    options: {},
    position: 0,
    ...over,
  }) as Target;

const spec = (over: Partial<ActionSpec> = {}): ActionSpec =>
  ({
    id: "zip",
    name: "Zip",
    description: "",
    icon: "",
    category: "",
    events: [],
    accepts: [],
    options: [],
    multi: false,
    ...over,
  }) as ActionSpec;

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(backend.tasks.list).mockResolvedValue([] as never);
  vi.mocked(backend.dropBar.list).mockResolvedValue([] as never);
  vi.mocked(backend.shares.list).mockResolvedValue([] as never);
});

describe("GridPanel", () => {
  it("clicking a runnable tile invokes the run backend method (not config)", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.grid.list).mockResolvedValue([
      target({ id: "run-1", label: "Run Me" }),
    ] as never);
    vi.mocked(backend.actions.specs).mockResolvedValue([
      spec({ id: "zip", events: ["clicked"] }),
    ] as never);

    render(<GridPanel onOpenSettings={vi.fn()} />);
    const tile = (await screen.findByText("Run Me")).closest("button") as HTMLElement;
    await user.click(tile);

    expect(backend.click).toHaveBeenCalledWith("run-1");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("clicking a config-only tile opens its config dialog instead of running it", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.grid.list).mockResolvedValue([
      target({ id: "cfg-1", actionId: "email", label: "Send Mail" }),
    ] as never);
    vi.mocked(backend.actions.specs).mockResolvedValue([
      spec({
        id: "email",
        events: ["dragged"],
        options: [{ key: "to", label: "To", type: "text" }],
      }),
    ] as never);

    render(<GridPanel onOpenSettings={vi.fn()} />);
    const tile = (await screen.findByText("Send Mail")).closest("button") as HTMLElement;
    await user.click(tile);

    expect(await screen.findByText("Edit Send Mail")).toBeInTheDocument();
    expect(backend.click).not.toHaveBeenCalled();
  });

  it("does nothing when clicking a drag-only tile with no options", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.grid.list).mockResolvedValue([
      target({ id: "none-1", actionId: "folder-watch", label: "Watch Only" }),
    ] as never);
    vi.mocked(backend.actions.specs).mockResolvedValue([
      spec({ id: "folder-watch", events: ["dragged"], options: [] }),
    ] as never);

    render(<GridPanel onOpenSettings={vi.fn()} />);
    const tile = (await screen.findByText("Watch Only")).closest("button") as HTMLElement;
    await user.click(tile);

    expect(backend.click).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("Option-held delete mode jiggles tiles and removes one via its X badge", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.grid.list).mockResolvedValue([
      target({ id: "del-1", label: "Deletable" }),
    ] as never);
    vi.mocked(backend.actions.specs).mockResolvedValue([
      spec({ id: "zip", events: ["clicked"] }),
    ] as never);

    render(<GridPanel onOpenSettings={vi.fn()} />);
    const tileButton = (await screen.findByText("Deletable")).closest("button") as HTMLElement;

    expect(screen.queryByTitle("Remove from Grid")).not.toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Alt", altKey: true });

    const badge = await screen.findByTitle("Remove from Grid");
    expect(tileButton.className).toContain("dz-jiggle");

    await user.click(badge);
    expect(backend.grid.remove).toHaveBeenCalledWith("del-1");
    expect(backend.click).not.toHaveBeenCalled();
  });
});
