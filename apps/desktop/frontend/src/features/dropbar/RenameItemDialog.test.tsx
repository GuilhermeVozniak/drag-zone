import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RenameItemDialog } from "@/features/dropbar/RenameItemDialog";
import { backend, type DropBarItem } from "@/lib/backend";

vi.mock("@/lib/backend");

const item = { id: "i1" } as DropBarItem;

beforeEach(() => vi.clearAllMocks());

describe("RenameItemDialog", () => {
  it("is closed when value is null", () => {
    render(<RenameItemDialog item={item} value={null} onValueChange={vi.fn()} />);
    expect(screen.queryByText("Rename Item")).not.toBeInTheDocument();
  });

  it("renames on Save and closes", async () => {
    const user = userEvent.setup();
    const onValueChange = vi.fn();
    render(<RenameItemDialog item={item} value={"notes"} onValueChange={onValueChange} />);
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(backend.dropBar.rename).toHaveBeenCalledWith("i1", "notes");
    expect(onValueChange).toHaveBeenCalledWith(null);
  });

  it("renames on Enter", async () => {
    const user = userEvent.setup();
    render(<RenameItemDialog item={item} value={"notes"} onValueChange={vi.fn()} />);
    await user.type(screen.getByRole("textbox"), "{Enter}");
    expect(backend.dropBar.rename).toHaveBeenCalledWith("i1", "notes");
  });

  it("Reset commits an empty label so the content-derived name returns", async () => {
    const user = userEvent.setup();
    const onValueChange = vi.fn();
    render(<RenameItemDialog item={item} value={"notes"} onValueChange={onValueChange} />);
    await user.click(screen.getByRole("button", { name: "Reset" }));
    expect(backend.dropBar.rename).toHaveBeenCalledWith("i1", "");
    expect(onValueChange).toHaveBeenCalledWith(null);
  });
});
