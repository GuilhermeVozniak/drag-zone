import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ConsoleDrawer } from "@/features/tasks/ConsoleDrawer";
import { __fireEvent, backend } from "@/lib/backend";

vi.mock("@/lib/backend");

const line = (text: string) => ({ line: text, at: "2026-01-01T00:00:00Z" });

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(backend.console.lines).mockResolvedValue(undefined as never);
});

describe("ConsoleDrawer", () => {
  it("shows the empty placeholder when there is no output", async () => {
    render(<ConsoleDrawer onClose={() => {}} />);
    expect(await screen.findByText("No script output yet.")).toBeInTheDocument();
  });

  it("loads the existing console buffer on mount", async () => {
    vi.mocked(backend.console.lines).mockResolvedValue([line("first"), line("second")] as never);
    render(<ConsoleDrawer onClose={() => {}} />);
    expect(await screen.findByText("first")).toBeInTheDocument();
    expect(screen.getByText("second")).toBeInTheDocument();
  });

  it("streams new lines from console:changed events", async () => {
    render(<ConsoleDrawer onClose={() => {}} />);
    // Wait out the mount-time buffer load so its resolution doesn't
    // clobber the streamed state.
    await screen.findByText("No script output yet.");
    act(() => __fireEvent("console:changed", [line("streamed")]));
    expect(await screen.findByText("streamed")).toBeInTheDocument();
  });

  it("survives a null event payload (Go marshals empty slices as null)", async () => {
    render(<ConsoleDrawer onClose={() => {}} />);
    act(() => __fireEvent("console:changed", null));
    await waitFor(() => expect(screen.getByText("No script output yet.")).toBeInTheDocument());
  });

  it("clears the console via the trash button", async () => {
    const user = userEvent.setup();
    render(<ConsoleDrawer onClose={() => {}} />);
    await user.click(screen.getByTitle("Clear console"));
    expect(backend.console.clear).toHaveBeenCalled();
  });

  it("closes via the X button", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<ConsoleDrawer onClose={onClose} />);
    await user.click(screen.getByTitle("Close console"));
    expect(onClose).toHaveBeenCalled();
  });
});
