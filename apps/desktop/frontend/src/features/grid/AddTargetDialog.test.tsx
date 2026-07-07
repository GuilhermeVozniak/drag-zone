import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AddTargetDialog } from "@/features/grid/AddTargetDialog";
import { type ActionSpec, backend, type Target } from "@/lib/backend";

vi.mock("@/lib/backend");

const spec = (over: Partial<ActionSpec>): ActionSpec =>
  ({
    id: "zip",
    name: "Zip",
    description: "Compress",
    icon: "archive",
    options: [],
    ...over,
  }) as ActionSpec;

beforeEach(() => vi.clearAllMocks());

describe("AddTargetDialog", () => {
  it("shows the catalogue, then the config form after picking an action", async () => {
    const user = userEvent.setup();
    render(
      <AddTargetDialog
        open
        onOpenChange={vi.fn()}
        specs={[spec({ id: "zip", name: "Zip" }), spec({ id: "trash", name: "Trash" })]}
      />,
    );
    expect(screen.getByText("Add to Grid")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Zip/ }));
    expect(screen.getByText("Name in grid")).toBeInTheDocument();
  });

  it("adds a new target with the action id, label, and option map", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    render(<AddTargetDialog open onOpenChange={onOpenChange} specs={[spec({})]} />);
    await user.click(screen.getByRole("button", { name: /Zip/ }));
    await user.click(screen.getByRole("button", { name: "Add to Grid" }));
    await waitFor(() => expect(backend.grid.add).toHaveBeenCalledWith("zip", "Zip", {}));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("updates an existing target in edit mode", async () => {
    const user = userEvent.setup();
    const editing = {
      id: "t1",
      actionId: "zip",
      label: "My Zip",
      options: {},
      shortcut: "",
    } as Target;
    render(<AddTargetDialog open onOpenChange={vi.fn()} specs={[spec({})]} editing={editing} />);
    expect(screen.getByText("Edit My Zip")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(backend.grid.update).toHaveBeenCalledWith({
        ...editing,
        label: "My Zip",
        options: {},
        shortcut: "",
      }),
    );
  });

  it("disables submit until every required option is filled", async () => {
    const user = userEvent.setup();
    const urlSpec = spec({
      id: "shorten-url",
      name: "Shorten",
      options: [{ key: "url", label: "URL", type: "text", required: true } as never],
    });
    render(<AddTargetDialog open onOpenChange={vi.fn()} specs={[urlSpec]} />);
    await user.click(screen.getByRole("button", { name: /Shorten/ }));
    const submit = screen.getByRole("button", { name: "Add to Grid" });
    expect(submit).toBeDisabled();
    await user.type(screen.getAllByRole("textbox")[2], "https://x");
    expect(submit).toBeEnabled();
  });
});
