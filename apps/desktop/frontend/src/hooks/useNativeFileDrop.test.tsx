import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useNativeFileDrop } from "@/hooks/useNativeFileDrop";
import { backend, type DropBarItem } from "@/lib/backend";
import { getDraggingDropBarItem, setDraggingDropBarItem, setUIScale } from "@/lib/dnd";
import { __emitFileDrop, __resetRuntimeStub } from "@/test/stubs/runtime";

vi.mock("@/lib/backend");

beforeEach(() => {
  __resetRuntimeStub();
  setUIScale(1);
  setDraggingDropBarItem(null);
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

  describe("combining onto a sibling Drop Bar item", () => {
    const items: DropBarItem[] = [
      {
        id: "target1",
        kind: "files",
        paths: ["/a.txt"],
        label: "a.txt",
        locked: false,
      } as DropBarItem,
      {
        id: "source1",
        kind: "files",
        paths: ["/b.txt"],
        label: "b.txt",
        locked: false,
      } as DropBarItem,
    ];

    it("combines when a drag-out is in flight and lands on another item's tile", () => {
      renderHook(() => useNativeFileDrop(items));
      setDraggingDropBarItem("source1");
      drop("target1", ["/b.txt"]);
      expect(backend.dropBar.combine).toHaveBeenCalledWith("target1", "source1");
      expect(backend.dropBar.add).not.toHaveBeenCalled();
    });

    it("consumes the drag-out source so a later drop can't reuse it", () => {
      renderHook(() => useNativeFileDrop(items));
      setDraggingDropBarItem("source1");
      drop("target1", ["/b.txt"]);
      expect(getDraggingDropBarItem()).toBeNull();
    });

    it("falls back to stashing a new item when no drag-out is in flight", () => {
      renderHook(() => useNativeFileDrop(items));
      drop("target1", ["/external.txt"]);
      expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: "files", paths: ["/external.txt"] });
      expect(backend.dropBar.combine).not.toHaveBeenCalled();
    });

    it("does not combine an item with itself", () => {
      renderHook(() => useNativeFileDrop(items));
      setDraggingDropBarItem("target1");
      drop("target1", ["/a.txt"]);
      expect(backend.dropBar.combine).not.toHaveBeenCalled();
      expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: "files", paths: ["/a.txt"] });
    });
  });
});
