import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

  it("single-click on a files item triggers Quick Look with its paths", async () => {
    const user = userEvent.setup();
    const item = filesItem({ paths: ["/unique/dropbartile/c.txt"], label: "c.txt" });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("c.txt").parentElement as HTMLElement;
    await user.click(tile);
    expect(backend.quickLook).toHaveBeenCalledWith(["/unique/dropbartile/c.txt"]);
  });

  it("single-click on a stack Quick Looks every path in it", async () => {
    const user = userEvent.setup();
    const item = filesItem({
      paths: ["/unique/dropbartile/s1.png", "/unique/dropbartile/s2.png"],
      label: "2 Items",
    });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("2 Items").parentElement as HTMLElement;
    await user.click(tile);
    expect(backend.quickLook).toHaveBeenCalledWith([
      "/unique/dropbartile/s1.png",
      "/unique/dropbartile/s2.png",
    ]);
  });

  it("does not Quick Look a non-files item on click", async () => {
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
    await user.click(tile);
    expect(backend.quickLook).not.toHaveBeenCalled();
  });

  it("does not Quick Look when a press turned into a drag-out", () => {
    const item = filesItem({ paths: ["/unique/dropbartile/drag.txt"], label: "drag.txt" });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("drag.txt").parentElement as HTMLElement;
    fireEvent.mouseDown(tile, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.mouseMove(tile, { clientX: 20, clientY: 0 });
    fireEvent.mouseUp(tile);
    fireEvent.click(tile);
    expect(backend.dragOut).toHaveBeenCalledWith("i1");
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

  it("clicking a fanned stack thumbnail opens that exact file in the default app", async () => {
    (backend.fileIcon as ReturnType<typeof vi.fn>).mockResolvedValue("QQ==");
    const user = userEvent.setup();
    const item = filesItem({
      paths: ["/unique/dropbartile/stack-front.png", "/unique/dropbartile/stack-back.png"],
      label: "2 Items",
      id: "stack1",
    });
    const { container } = render(<DropBarTile item={item} onRemove={vi.fn()} />);
    await waitFor(() => expect(container.querySelectorAll("img")).toHaveLength(2));
    const imgs = container.querySelectorAll("img");

    // paths[0] is drawn on top, i.e. last in DOM order (see StackFan).
    await user.click(imgs[1]);
    expect(backend.openPath).toHaveBeenCalledWith("/unique/dropbartile/stack-front.png");
    await user.click(imgs[0]);
    expect(backend.openPath).toHaveBeenCalledWith("/unique/dropbartile/stack-back.png");

    // A plain click on a thumbnail must not start a drag-out or Quick Look.
    expect(backend.dragOut).not.toHaveBeenCalled();
    expect(backend.quickLook).not.toHaveBeenCalled();
  });

  it("mousedown+move on a fanned thumbnail still starts a native drag-out", async () => {
    (backend.fileIcon as ReturnType<typeof vi.fn>).mockResolvedValue("QQ==");
    const item = filesItem({
      paths: ["/unique/dropbartile/stack-drag-a.png", "/unique/dropbartile/stack-drag-b.png"],
      label: "2 Items",
      id: "stack2",
    });
    const { container } = render(<DropBarTile item={item} onRemove={vi.fn()} />);
    await waitFor(() => expect(container.querySelectorAll("img")).toHaveLength(2));
    const imgs = container.querySelectorAll("img");
    fireEvent.mouseDown(imgs[0], { button: 0, clientX: 0, clientY: 0 });
    fireEvent.mouseMove(imgs[0], { clientX: 20, clientY: 0 });
    expect(backend.dragOut).toHaveBeenCalledWith("stack2");
  });

  it("highlights the tile while a combinable Drop Bar item drags over it", () => {
    const item = filesItem({ paths: ["/unique/dropbartile/target.txt"], label: "target.txt" });
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("target.txt").parentElement as HTMLElement;
    fireEvent.dragEnter(tile);
    expect(tile.className).toContain("ring-2");
    fireEvent.dragLeave(tile);
    expect(tile.className).not.toContain("ring-2");
  });

  it("does not highlight a non-files tile on drag-over (only stacks of files combine)", () => {
    const item = {
      id: "i5",
      kind: "url",
      text: "https://y.test",
      label: "y.test",
      locked: false,
    } as DropBarItem;
    render(<DropBarTile item={item} onRemove={vi.fn()} />);
    const tile = screen.getByText("y.test").parentElement as HTMLElement;
    fireEvent.dragEnter(tile);
    expect(tile.className).not.toContain("ring-2");
  });
});
