import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AddonsTab } from "@/features/settings/AddonsTab";
import { type AddonInfo, backend } from "@/lib/backend";

vi.mock("@/lib/backend");

const addon = (over: Partial<AddonInfo>): AddonInfo => ({ name: "", installed: false, ...over });

beforeEach(() => vi.clearAllMocks());

describe("AddonsTab", () => {
  it("lists the add-on catalogue from the backend", async () => {
    vi.mocked(backend.addons.list).mockResolvedValue([
      addon({ name: "Zip It" }),
      addon({ name: "Resize Images" }),
    ] as never);
    render(<AddonsTab />);
    await waitFor(() => expect(screen.getByText("Zip It")).toBeInTheDocument());
    expect(screen.getByText("Resize Images")).toBeInTheDocument();
  });

  it("installs an add-on when its Install button is clicked", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.addons.list).mockResolvedValue([addon({ name: "Zip It" })] as never);
    vi.mocked(backend.addons.install).mockResolvedValue(undefined as never);
    render(<AddonsTab />);
    await waitFor(() => expect(screen.getByText("Zip It")).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "Install" }));
    expect(backend.addons.install).toHaveBeenCalledWith("Zip It");
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Installed" })).toBeInTheDocument(),
    );
  });

  it("shows already-installed add-ons as disabled", async () => {
    vi.mocked(backend.addons.list).mockResolvedValue([
      addon({ name: "Resize Images", installed: true }),
    ] as never);
    render(<AddonsTab />);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Installed" })).toBeInTheDocument(),
    );
    expect(screen.getByRole("button", { name: "Installed" })).toBeDisabled();
    expect(backend.addons.install).not.toHaveBeenCalled();
  });

  it("surfaces a failed install as an inline error", async () => {
    const user = userEvent.setup();
    vi.mocked(backend.addons.list).mockResolvedValue([addon({ name: "Zip It" })] as never);
    vi.mocked(backend.addons.install).mockRejectedValue(new Error("network down") as never);
    render(<AddonsTab />);
    await waitFor(() => expect(screen.getByText("Zip It")).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "Install" }));
    await waitFor(() => expect(screen.getByText(/network down/)).toBeInTheDocument());
  });
});
