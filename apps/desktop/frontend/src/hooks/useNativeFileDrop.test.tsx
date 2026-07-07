import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useNativeFileDrop } from "@/hooks/useNativeFileDrop";
import { backend } from "@/lib/backend";
import { setUIScale } from "@/lib/dnd";
import { __emitFileDrop, __resetRuntimeStub } from "@/test/stubs/runtime";

vi.mock("@/lib/backend");

beforeEach(() => {
  __resetRuntimeStub();
  setUIScale(1);
  document.body.innerHTML = "";
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

function drop(dropId: string | null, paths: string[]) {
  if (dropId) {
    document.body.innerHTML = `<div data-drop-id="${dropId}"></div>`;
    vi.spyOn(document, "elementFromPoint").mockReturnValue(
      document.querySelector("[data-drop-id]"),
    );
  } else {
    vi.spyOn(document, "elementFromPoint").mockReturnValue(null);
  }
  __emitFileDrop(10, 10, paths);
}

describe("useNativeFileDrop", () => {
  it("stashes files in the drop bar for the dropbar target", () => {
    renderHook(() => useNativeFileDrop());
    drop("dropbar", ["/a.txt"]);
    expect(backend.playDropSound).toHaveBeenCalled();
    expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: "files", paths: ["/a.txt"] });
  });
  it("adds files to the grid for the add-to-grid target", () => {
    renderHook(() => useNativeFileDrop());
    drop("add-to-grid", ["/a", "/b"]);
    expect(backend.grid.addFromPaths).toHaveBeenCalledWith(["/a", "/b"]);
  });
  it("runs the action and hides the window for a normal target", () => {
    renderHook(() => useNativeFileDrop());
    drop("t123", ["/a.txt"]);
    expect(backend.drop).toHaveBeenCalledWith("t123", { kind: "files", paths: ["/a.txt"] });
    expect(backend.window.hide).toHaveBeenCalled();
  });
  it("ignores drops with no target under the cursor", () => {
    renderHook(() => useNativeFileDrop());
    drop(null, ["/a"]);
    expect(backend.playDropSound).not.toHaveBeenCalled();
  });
  it("ignores empty drops", () => {
    renderHook(() => useNativeFileDrop());
    drop("t123", []);
    expect(backend.playDropSound).not.toHaveBeenCalled();
  });
});
