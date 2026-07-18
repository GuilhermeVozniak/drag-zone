import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CommandLineTab } from "@/features/settings/CommandLineTab";
import { backend } from "@/lib/backend";

vi.mock("@/lib/backend");

beforeEach(() => vi.clearAllMocks());

describe("CommandLineTab", () => {
  it("offers to install the CLI when not installed", async () => {
    vi.mocked(backend.cli.installed).mockResolvedValue(false as never);
    render(<CommandLineTab />);
    await waitFor(() => expect(screen.getByText("Not installed")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Install Command Line Tool" })).toBeInTheDocument();
  });

  it("installs the CLI and reflects the installed state afterwards", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.cli.installed)
      .mockResolvedValueOnce(false as never)
      .mockResolvedValueOnce(true as never);
    vi.mocked(backend.cli.install).mockResolvedValue(undefined as never);
    render(<CommandLineTab />);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Install Command Line Tool" })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("button", { name: "Install Command Line Tool" }));
    expect(backend.cli.install).toHaveBeenCalledTimes(1);
    await waitFor(() =>
      expect(screen.getByText("Installed at /usr/local/bin/dz")).toBeInTheDocument(),
    );
    expect(screen.queryByRole("button", { name: "Install Command Line Tool" })).toBeNull();
  });

  it("shows the installed path directly when already installed", async () => {
    vi.mocked(backend.cli.installed).mockResolvedValue(true as never);
    render(<CommandLineTab />);
    await waitFor(() =>
      expect(screen.getByText("Installed at /usr/local/bin/dz")).toBeInTheDocument(),
    );
    expect(screen.queryByRole("button", { name: "Install Command Line Tool" })).toBeNull();
  });

  it("surfaces an install error", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.cli.installed).mockResolvedValue(false as never);
    vi.mocked(backend.cli.install).mockRejectedValue(new Error("permission denied") as never);
    render(<CommandLineTab />);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Install Command Line Tool" })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("button", { name: "Install Command Line Tool" }));
    await waitFor(() => expect(screen.getByText(/permission denied/)).toBeInTheDocument());
  });
});
