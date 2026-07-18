import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DevelopActionRow } from "@/features/settings/DevelopActionRow";
import { backend } from "@/lib/backend";

vi.mock("@/lib/backend");

beforeEach(() => vi.clearAllMocks());

describe("DevelopActionRow", () => {
  it("disables Develop until a name is entered", async () => {
    render(<DevelopActionRow />);
    expect(screen.getByRole("button", { name: "Develop" })).toBeDisabled();
  });

  it("generates a Ruby template by default", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.actions.develop).mockResolvedValue(undefined as never);
    render(<DevelopActionRow />);
    await user.type(screen.getByPlaceholderText("New action name"), "My Action");
    await user.click(screen.getByRole("button", { name: "Develop" }));
    expect(backend.actions.develop).toHaveBeenCalledWith("My Action", "ruby");
  });

  it("clears the name field after developing the action", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.actions.develop).mockResolvedValue(undefined as never);
    render(<DevelopActionRow />);
    const input = screen.getByPlaceholderText("New action name") as HTMLInputElement;
    await user.type(input, "My Action");
    await user.click(screen.getByRole("button", { name: "Develop" }));
    expect(input.value).toBe("");
  });

  it("generates a Python template when Python is selected", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.actions.develop).mockResolvedValue(undefined as never);
    render(<DevelopActionRow />);
    await user.type(screen.getByPlaceholderText("New action name"), "Thumbnailer");
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "Python" }));
    await waitFor(() => expect(screen.queryByRole("listbox")).not.toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "Develop" }));
    expect(backend.actions.develop).toHaveBeenCalledWith("Thumbnailer", "python");
  });
});
