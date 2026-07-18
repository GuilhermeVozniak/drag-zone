import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TargetTile } from "@/features/grid/TargetTile";
import { type ActionSpec, backend, type Target } from "@/lib/backend";

vi.mock("@/lib/backend");

const target = (over: Partial<Target> = {}): Target =>
  ({
    id: "t1",
    actionId: "zip",
    label: "Zip It",
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

function renderTile(over: {
  target?: Target;
  spec?: ActionSpec | undefined;
  showKeyOverlay?: boolean;
  optionHeld?: boolean;
}) {
  const handlers = {
    onClick: vi.fn(),
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onRemove: vi.fn(),
    onDropBarItemDrop: vi.fn(),
    onTextDrop: vi.fn(),
    onReorder: vi.fn(),
  };
  render(
    <TargetTile
      target={over.target ?? target()}
      spec={over.spec ?? spec()}
      tilePx={80}
      iconPx={64}
      showKeyOverlay={over.showKeyOverlay ?? true}
      optionHeld={over.optionHeld ?? false}
      {...handlers}
    />,
  );
  return handlers;
}

beforeEach(() => vi.clearAllMocks());

describe("TargetTile", () => {
  it("renders the action icon, label, and single-key shortcut", () => {
    renderTile({ target: target({ label: "My Zip", shortcut: "z" }) });
    expect(screen.getByText("My Zip")).toBeInTheDocument();
    expect(screen.getByText("Z")).toBeInTheDocument();
    const tile = screen.getByText("My Zip").closest("button") as HTMLElement;
    expect(tile.querySelector("svg")).toBeTruthy();
  });

  it("hides the shortcut overlay when showKeyOverlay is false", () => {
    renderTile({ target: target({ label: "Zip", shortcut: "z" }), showKeyOverlay: false });
    expect(screen.queryByText("Z")).not.toBeInTheDocument();
  });

  it("shows the key-modifier glyph only while a drag hovers the tile", () => {
    renderTile({
      target: target({ label: "Email It" }),
      spec: spec({ keyModifier: "option" }),
    });
    expect(screen.queryByText("⌥")).not.toBeInTheDocument();

    const tile = screen.getByText("Email It").closest("button") as HTMLElement;
    fireEvent.dragOver(tile);
    const glyph = screen.getByText("⌥");
    expect(glyph).toBeInTheDocument();
    expect(glyph).toHaveAttribute("title", "Hold to change behavior on drop");

    fireEvent.dragLeave(tile);
    expect(screen.queryByText("⌥")).not.toBeInTheDocument();
  });

  it("does not show a modifier glyph while hovering an action with no keyModifier", () => {
    renderTile({ target: target({ label: "Plain Zip" }), spec: spec({ keyModifier: undefined }) });
    const tile = screen.getByText("Plain Zip").closest("button") as HTMLElement;
    fireEvent.dragOver(tile);
    expect(screen.queryByText("⌥")).not.toBeInTheDocument();
    expect(screen.queryByText("⌘")).not.toBeInTheDocument();
  });

  it("shows the delete-mode X badge when optionHeld and calls onRemove when clicked", async () => {
    const user = userEvent.setup();
    const handlers = renderTile({
      target: target({ id: "del-1", label: "Deletable" }),
      optionHeld: true,
    });
    const badge = screen.getByTitle("Remove from Grid");
    expect(badge).toBeInTheDocument();
    await user.click(badge);
    expect(handlers.onRemove).toHaveBeenCalledTimes(1);
    // Clicking the badge must not also trigger the tile's own onClick.
    expect(handlers.onClick).not.toHaveBeenCalled();
    expect(backend.click).not.toHaveBeenCalled();
  });
});
