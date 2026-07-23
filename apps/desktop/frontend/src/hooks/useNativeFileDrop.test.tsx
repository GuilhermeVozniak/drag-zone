import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useNativeFileDrop } from "@/hooks/useNativeFileDrop";
import { __fireEvent, __resetBackendMock } from "@/lib/__mocks__/backend";
import { backend, type DropBarItem } from "@/lib/backend";
import {
  __resetNativeFileDropForTests,
  getDraggingDropBarItem,
  setDraggingDropBarItem,
  setUIScale,
} from "@/lib/dnd";
import { __emitFileDrop, __resetRuntimeStub } from "@/test/stubs/runtime";

vi.mock("@/lib/backend");

beforeEach(() => {
  __resetRuntimeStub();
  __resetNativeFileDropForTests();
  __resetBackendMock();
  setUIScale(1);
  setDraggingDropBarItem(null);
  document.body.innerHTML = "";
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

function drop(dropId: string | null, paths: string[], opts?: { x?: number; rect?: DOMRect }) {
  if (dropId) {
    document.body.innerHTML = `<div data-drop-id="${dropId}"></div>`;
    const el = document.querySelector("[data-drop-id]") as HTMLElement;
    if (opts?.rect) {
      vi.spyOn(el, "getBoundingClientRect").mockReturnValue(opts.rect);
    }
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([el]);
  } else {
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([]);
  }
  __emitFileDrop(opts?.x ?? 10, 10, paths);
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
  it("runs the action and hides the window for a normal target", async () => {
    renderHook(() => useNativeFileDrop());
    drop("t123", ["/a.txt"]);
    expect(backend.drop).toHaveBeenCalledWith("t123", { kind: "files", paths: ["/a.txt"] });
    // The window hides once the drop binding resolves.
    await vi.waitFor(() => expect(backend.window.hide).toHaveBeenCalled());
  });
  it("keeps the window open and reports when the drop fails", async () => {
    vi.mocked(backend.drop).mockRejectedValueOnce(new Error("no such target"));
    renderHook(() => useNativeFileDrop());
    drop("t123", ["/a.txt"]);
    await Promise.resolve();
    await Promise.resolve();
    expect(backend.window.hide).not.toHaveBeenCalled();
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

    it("stacks external files dropped onto the center of an existing stack", async () => {
      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new1" } as DropBarItem);
      renderHook(() => useNativeFileDrop(items));
      // jsdom rects are zero-sized, which dropZone reads as "center".
      drop("target1", ["/external.txt"]);
      expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: "files", paths: ["/external.txt"] });
      await vi.waitFor(() =>
        expect(backend.dropBar.combine).toHaveBeenCalledWith("target1", "new1"),
      );
      expect(backend.dropBar.move).not.toHaveBeenCalled();
    });

    it("stashes a new item at the tile's spot when dropped on its edge", async () => {
      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new1" } as DropBarItem);
      renderHook(() => useNativeFileDrop(items));
      const rect = { left: 0, width: 100 } as DOMRect;
      // target1 is at index 0: before -> insert at 0, after -> insert at 1.
      drop("target1", ["/external.txt"], { x: 10, rect });
      await vi.waitFor(() => expect(backend.dropBar.move).toHaveBeenCalledWith("new1", 0));
      expect(backend.dropBar.combine).not.toHaveBeenCalled();

      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new2" } as DropBarItem);
      drop("target1", ["/other.txt"], { x: 90, rect });
      await vi.waitFor(() => expect(backend.dropBar.move).toHaveBeenCalledWith("new2", 1));
    });

    it("never merges external files into a text/URL item, even on its center", async () => {
      const textItems: DropBarItem[] = [
        {
          id: "text1",
          kind: "text",
          text: "hello",
          label: "hello",
          locked: false,
        } as DropBarItem,
      ];
      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new1" } as DropBarItem);
      renderHook(() => useNativeFileDrop(textItems));
      drop("text1", ["/external.txt"]);
      expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: "files", paths: ["/external.txt"] });
      await Promise.resolve();
      await Promise.resolve();
      expect(backend.dropBar.combine).not.toHaveBeenCalled();
      expect(backend.dropBar.move).not.toHaveBeenCalled();
    });

    it("does not combine an item with itself", async () => {
      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new1" } as DropBarItem);
      renderHook(() => useNativeFileDrop(items));
      setDraggingDropBarItem("target1");
      drop("target1", ["/a.txt"]);
      expect(backend.dropBar.add).toHaveBeenCalledWith({ kind: "files", paths: ["/a.txt"] });
      await Promise.resolve();
      await Promise.resolve();
      expect(backend.dropBar.combine).not.toHaveBeenCalled();
      expect(backend.dropBar.move).not.toHaveBeenCalled();
    });

    // Regression coverage for the data-loss bug: a stale drag-out source
    // must never survive past the drop that (mis)handles it, so it can't
    // corrupt a later, unrelated drop.
    it("clears the drag-out source after ANY drop, even one that isn't onto a sibling tile", async () => {
      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new1" } as DropBarItem);
      renderHook(() => useNativeFileDrop(items));
      setDraggingDropBarItem("source1");
      // The drag-out resolves onto the Drop Bar's own background, not a
      // sibling tile (e.g. the user let go over the bar itself).
      drop("dropbar", ["/external.txt"]);
      expect(getDraggingDropBarItem()).toBeNull();

      // A second, completely unrelated native drop onto a Drop Bar tile
      // must combine only the newly added item, never the stale source id.
      drop("target1", ["/unrelated.txt"]);
      expect(backend.dropBar.add).toHaveBeenCalledWith({
        kind: "files",
        paths: ["/unrelated.txt"],
      });
      await vi.waitFor(() =>
        expect(backend.dropBar.combine).toHaveBeenCalledWith("target1", "new1"),
      );
      expect(backend.dropBar.combine).not.toHaveBeenCalledWith("target1", "source1");
    });

    it("clears the drag-out source when the grid-target and add-to-grid drops consume it", () => {
      renderHook(() => useNativeFileDrop(items));

      setDraggingDropBarItem("source1");
      drop("add-to-grid", ["/a"]);
      expect(getDraggingDropBarItem()).toBeNull();

      setDraggingDropBarItem("source1");
      drop("t123", ["/a"]);
      expect(getDraggingDropBarItem()).toBeNull();
    });

    it("clears the drag-out source when the native drag session ends without a matching drop", async () => {
      vi.mocked(backend.dropBar.add).mockResolvedValue({ id: "new1" } as DropBarItem);
      renderHook(() => useNativeFileDrop(items));
      setDraggingDropBarItem("source1");
      // Simulate the drag-out resolving to Finder (or being cancelled):
      // OnFileDrop never fires in that case, but the backend still signals
      // that the drag session is over.
      __fireEvent("dropbar:dragended");
      expect(getDraggingDropBarItem()).toBeNull();

      // A later, unrelated drop onto a sibling tile must not reuse the
      // stale source: it combines only the newly added item.
      drop("target1", ["/unrelated.txt"]);
      expect(backend.dropBar.add).toHaveBeenCalledWith({
        kind: "files",
        paths: ["/unrelated.txt"],
      });
      await vi.waitFor(() =>
        expect(backend.dropBar.combine).toHaveBeenCalledWith("target1", "new1"),
      );
      expect(backend.dropBar.combine).not.toHaveBeenCalledWith("target1", "source1");
    });
  });
});
