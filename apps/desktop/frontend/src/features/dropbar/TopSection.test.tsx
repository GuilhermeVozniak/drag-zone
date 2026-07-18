import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TopSection } from "@/features/dropbar/TopSection";
import type { DropBarItem } from "@/lib/backend";

vi.mock("@/lib/backend");

const item = (over: Partial<DropBarItem>): DropBarItem =>
  ({
    id: over.id as string,
    kind: "files",
    paths: [`/unique/topsection/${over.id}.txt`],
    label: over.id as string,
    locked: false,
    ...over,
  }) as DropBarItem;

beforeEach(() => vi.clearAllMocks());

describe("TopSection", () => {
  it("renders the stashed items", () => {
    render(
      <TopSection items={[item({ id: "one" }), item({ id: "two" }), item({ id: "three" })]} />,
    );
    expect(screen.getByText("one")).toBeInTheDocument();
    expect(screen.getByText("two")).toBeInTheDocument();
    expect(screen.getByText("three")).toBeInTheDocument();
  });

  it("renders every item inside a scrollable overflow container, however many there are", () => {
    const many = Array.from({ length: 20 }, (_, i) => item({ id: `stashed-${i}` }));
    render(<TopSection items={many} />);
    for (const it of many) {
      expect(screen.getByText(it.label)).toBeInTheDocument();
    }
    const container = screen.getByText("stashed-0").closest("[data-drop-id='dropbar']");
    expect(container).not.toBeNull();
    expect(container?.className).toContain("overflow-y-auto");
    expect(container?.className).toContain("max-h-[184px]");
  });

  it("renders the persistent Add to Grid and Drop Bar tiles, and wires the Add click", async () => {
    const user = userEvent.setup();
    const onAddClick = vi.fn();
    render(<TopSection items={[]} onAddClick={onAddClick} />);
    expect(screen.getByText("Add to Grid")).toBeInTheDocument();
    expect(screen.getByText("Drop Bar")).toBeInTheDocument();
    await user.click(screen.getByText("Add to Grid"));
    expect(onAddClick).toHaveBeenCalledTimes(1);
  });

  it("omits the Add to Grid tile when showAddToGrid is false (popped-out bar)", () => {
    render(<TopSection items={[]} showAddToGrid={false} />);
    expect(screen.queryByText("Add to Grid")).not.toBeInTheDocument();
    expect(screen.getByText("Drop Bar")).toBeInTheDocument();
  });
});
