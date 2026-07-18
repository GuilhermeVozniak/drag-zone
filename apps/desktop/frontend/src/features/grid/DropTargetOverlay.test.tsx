import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DropTargetOverlay } from "./DropTargetOverlay";

describe("DropTargetOverlay", () => {
  it("renders the drop affordance and stays pointer-events-none when active", () => {
    render(<DropTargetOverlay active={true} />);
    const overlay = screen.getByText("Drop to add").closest("div[aria-hidden]");
    expect(overlay).not.toBeNull();
    expect(overlay).toHaveAttribute("data-state", "active");
    expect(overlay?.className).toContain("pointer-events-none");
    expect(overlay?.className).toContain("opacity-100");
  });

  it("fades out (but stays mounted and non-interactive) when inactive", () => {
    render(<DropTargetOverlay active={false} />);
    const overlay = screen.getByText("Drop to add").closest("div[aria-hidden]");
    expect(overlay).not.toBeNull();
    expect(overlay).toHaveAttribute("data-state", "inactive");
    expect(overlay?.className).toContain("pointer-events-none");
    expect(overlay?.className).toContain("opacity-0");
  });
});
