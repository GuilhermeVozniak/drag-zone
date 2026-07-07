import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OptionsForm } from "@/features/grid/OptionsForm";
import { backend, type OptionField } from "@/lib/backend";

vi.mock("@/lib/backend");

const field = (over: Partial<OptionField>): OptionField =>
  ({ key: "k", label: "K", type: "text", ...over }) as OptionField;

beforeEach(() => vi.clearAllMocks());

describe("OptionsForm", () => {
  it("reports text edits through onChange, merged into the value map", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <OptionsForm
        fields={[field({ key: "name", label: "Name", type: "text" })]}
        values={{}}
        onChange={onChange}
      />,
    );
    await user.type(screen.getByRole("textbox"), "a");
    expect(onChange).toHaveBeenCalledWith({ name: "a" });
  });

  it('toggles a checkbox field to the string "true"', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <OptionsForm
        fields={[field({ key: "flag", label: "Flag", type: "checkbox" })]}
        values={{}}
        onChange={onChange}
      />,
    );
    await user.click(screen.getByRole("switch"));
    expect(onChange).toHaveBeenCalledWith({ flag: "true" });
  });

  it("runs the folder picker and stores the chosen path", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    vi.mocked(backend.dialogs.chooseFolder).mockResolvedValue("/picked/dir" as never);
    render(
      <OptionsForm
        fields={[field({ key: "dir", label: "Dir", type: "folder" })]}
        values={{}}
        onChange={onChange}
      />,
    );
    await user.click(screen.getByRole("button", { name: /choose/i }));
    await waitFor(() => expect(onChange).toHaveBeenCalledWith({ dir: "/picked/dir" }));
  });

  it("uses the application picker for an app field", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    vi.mocked(backend.dialogs.chooseApplication).mockResolvedValue("/Apps/X.app" as never);
    render(
      <OptionsForm
        fields={[field({ key: "app", label: "App", type: "app" })]}
        values={{}}
        onChange={onChange}
      />,
    );
    await user.click(screen.getByRole("button", { name: /choose/i }));
    await waitFor(() => expect(onChange).toHaveBeenCalledWith({ app: "/Apps/X.app" }));
    expect(backend.dialogs.chooseFolder).not.toHaveBeenCalled();
  });

  it("does not store a path when the picker is cancelled", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    vi.mocked(backend.dialogs.chooseFolder).mockResolvedValue("" as never);
    render(
      <OptionsForm
        fields={[field({ key: "dir", label: "Dir", type: "folder" })]}
        values={{}}
        onChange={onChange}
      />,
    );
    await user.click(screen.getByRole("button", { name: /choose/i }));
    await waitFor(() => expect(backend.dialogs.chooseFolder).toHaveBeenCalled());
    expect(onChange).not.toHaveBeenCalled();
  });
});
