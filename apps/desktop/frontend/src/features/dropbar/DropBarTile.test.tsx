import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DropBarTile } from "@/features/dropbar/DropBarTile";
import { backend, type DropBarItem } from "@/lib/backend";

vi.mock("@/lib/backend");

const filesItem = (over: Partial<DropBarItem> = {}): DropBarItem =>
  ({
    id: "i1",
    kind: "files",
    paths: ["/unique/dropbartile/a.txt"],
    label: "a.txt",
    locked: false,
    ...over,
  }) as DropBarItem;

beforeEach(() => vi.clearAllMocks());

describe("DropBarTile", () => {
  it("renders the item's label", () => {
    render(<DropBarTile item={filesItem({ label: "report.pdf" })} onRemove={vi.fn()} />);
    expect(screen.getByText("report.pdf")).toBeInTheDocument();
  });

  it("renders a stack count badge for multiple paths", () => {
    render(
      <DropBarTile
        item={filesItem({
          paths: ["/unique/dropbartile/b1.txt", "/unique/dropbartile/b2.txt"],
          label: "Stack",
        })}
        onRemove={vi.fn()}
      />,
    );
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("double-click on a files item triggers Quick Look with its paths", async () => {
    const user = userEvent.setup();
    const item = filesItem({ paths: ["/unique/dropbartile/c.txt"], label: "c.txt" });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("c.txt").parentElement as HTMLElement;
    await user.dblClick(tile);
    expect(backend.quickLook).toHaveBeenCalledWith(["/unique/dropbartile/c.txt"]);
  });

  it("does not Quick Look a non-files item on double-click", async () => {
    const user = userEvent.setup();
    const item = {
      id: "i2",
      kind: "text",
      text: "hello",
      label: "hello",
      locked: false,
    } as DropBarItem;
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("hello").parentElement as HTMLElement;
    await user.dblClick(tile);
    expect(backend.quickLook).not.toHaveBeenCalled();
  });

  it("mousedown then move past the threshold starts a native drag-out", () => {
    const item = filesItem({ paths: ["/unique/dropbartile/d.txt"], label: "d.txt" });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("d.txt").parentElement as HTMLElement;
    fireEvent.mouseDown(tile, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.mouseMove(tile, { clientX: 20, clientY: 0 });
    expect(backend.dragOut).toHaveBeenCalledWith("i1");
  });

  it("does not drag out on a move within the threshold", () => {
    const item = filesItem({ paths: ["/unique/dropbartile/e.txt"], label: "e.txt" });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("e.txt").parentElement as HTMLElement;
    fireEvent.mouseDown(tile, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.mouseMove(tile, { clientX: 2, clientY: 0 });
    expect(backend.dragOut).not.toHaveBeenCalled();
  });

  it("does not drag out a non-files item on mousedown+move", () => {
    const item = {
      id: "i3",
      kind: "url",
      text: "https://x.test",
      label: "x.test",
      locked: false,
    } as DropBarItem;
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("x.test").parentElement as HTMLElement;
    fireEvent.mouseDown(tile, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.mouseMove(tile, { clientX: 20, clientY: 0 });
    expect(backend.dragOut).not.toHaveBeenCalled();
  });

  it("calls onRemove when the X badge is clicked", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    const item = filesItem({ paths: ["/unique/dropbartile/f.txt"], label: "f.txt", id: "i4" });
    render(<DropBarTile item={item} onRemove={onRemove} />);
    const tile = screen.getByText("f.txt").parentElement as HTMLElement;
    const removeButton = tile.querySelector("button") as HTMLElement;
    await user.click(removeButton);
    expect(onRemove).toHaveBeenCalledWith("i4");
  });
});
