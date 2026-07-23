import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { reportError } from "@/lib/report";

vi.mock("sonner", () => ({
  toast: { error: vi.fn() },
}));

beforeEach(() => vi.clearAllMocks());

describe("reportError", () => {
  it("surfaces an Error's message", () => {
    reportError("Save failed", new Error("disk full"));
    expect(toast.error).toHaveBeenCalledWith("Save failed", { description: "disk full" });
  });

  it("stringifies non-Error rejections (Wails bindings reject with strings)", () => {
    reportError("Load failed", "permission denied");
    expect(toast.error).toHaveBeenCalledWith("Load failed", {
      description: "permission denied",
    });
  });
});
