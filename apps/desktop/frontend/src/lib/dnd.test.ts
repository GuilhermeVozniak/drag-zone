import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  __resetNativeFileDropForTests,
  initNativeFileDrop,
  payloadFromDataTransfer,
  reorderIndex,
  setUIScale,
} from "@/lib/dnd";
import { __emitFileDrop, __resetRuntimeStub } from "@/test/stubs/runtime";

function fakeDT(data: Record<string, string>): DataTransfer {
  return { getData: (t: string) => data[t] ?? "" } as unknown as DataTransfer;
}

beforeEach(() => {
  __resetRuntimeStub();
  __resetNativeFileDropForTests();
  setUIScale(1);
  document.body.innerHTML = "";
  vi.restoreAllMocks();
});

describe("payloadFromDataTransfer", () => {
  it("reads a url from the first non-comment uri-list line, trimmed", () => {
    const dt = fakeDT({ "text/uri-list": "# comment\nhttps://example.com  \n" });
    expect(payloadFromDataTransfer(dt)).toEqual({ kind: "url", text: "https://example.com" });
  });
  it("prefers a url over plain text when both are present", () => {
    const dt = fakeDT({ "text/uri-list": "https://a.test", "text/plain": "ignored" });
    expect(payloadFromDataTransfer(dt)).toEqual({ kind: "url", text: "https://a.test" });
  });
  it("falls back to plain text when there is no uri-list", () => {
    const dt = fakeDT({ "text/plain": "hello world" });
    expect(payloadFromDataTransfer(dt)).toEqual({ kind: "text", text: "hello world" });
  });
  it("returns null when the transfer carries neither url nor text", () => {
    expect(payloadFromDataTransfer(fakeDT({}))).toBeNull();
  });
  it("returns null when the uri-list holds only comment lines", () => {
    // no non-comment uri and no text/plain -> null
    expect(payloadFromDataTransfer(fakeDT({ "text/uri-list": "# only a comment" }))).toBeNull();
  });
});

describe("initNativeFileDrop", () => {
  it("resolves the drop-id from the element under the cursor", () => {
    document.body.innerHTML = `<div data-drop-id="dropbar"><span id="child"></span></div>`;
    const child = document.getElementById("child") as Element;
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([child]);
    const onFiles = vi.fn();
    initNativeFileDrop({ onFiles });
    __emitFileDrop(100, 200, ["/a.txt"]);
    expect(onFiles).toHaveBeenCalledWith("dropbar", ["/a.txt"], "center");
  });
  it("walks past see-through layers (drop overlay) to the tile beneath", () => {
    document.body.innerHTML = `
      <div id="tile" data-drop-id="t123"></div>
      <div id="overlay"></div>`;
    const overlay = document.getElementById("overlay") as Element;
    const tile = document.getElementById("tile") as Element;
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([overlay, tile]);
    const onFiles = vi.fn();
    initNativeFileDrop({ onFiles });
    __emitFileDrop(10, 10, ["/a.txt"]);
    expect(onFiles).toHaveBeenCalledWith("t123", ["/a.txt"], "center");
  });
  it("un-zooms the cursor coordinates by the UI scale before hit-testing", () => {
    setUIScale(2);
    const spy = vi.spyOn(document, "elementsFromPoint").mockReturnValue([]);
    initNativeFileDrop({ onFiles: vi.fn() });
    __emitFileDrop(100, 200, ["/a"]);
    expect(spy).toHaveBeenCalledWith(50, 100);
  });
  it("passes a null drop-id when nothing is under the cursor", () => {
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([]);
    const onFiles = vi.fn();
    initNativeFileDrop({ onFiles });
    __emitFileDrop(1, 1, ["/a"]);
    expect(onFiles).toHaveBeenCalledWith(null, ["/a"], "center");
  });
  it("classifies the drop zone from the cursor's position within the tile", () => {
    document.body.innerHTML = `<div data-drop-id="item1"></div>`;
    const el = document.querySelector("[data-drop-id]") as HTMLElement;
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([el]);
    vi.spyOn(el, "getBoundingClientRect").mockReturnValue({
      left: 100,
      width: 80,
    } as DOMRect);
    const onFiles = vi.fn();
    initNativeFileDrop({ onFiles });
    __emitFileDrop(110, 10, ["/a"]); // rel 0.125 -> before
    __emitFileDrop(140, 10, ["/a"]); // rel 0.5 -> center
    __emitFileDrop(170, 10, ["/a"]); // rel 0.875 -> after
    expect(onFiles).toHaveBeenNthCalledWith(1, "item1", ["/a"], "before");
    expect(onFiles).toHaveBeenNthCalledWith(2, "item1", ["/a"], "center");
    expect(onFiles).toHaveBeenNthCalledWith(3, "item1", ["/a"], "after");
  });
  it("dispatches to the latest handler after a remount (Wails ignores re-registration)", () => {
    vi.spyOn(document, "elementsFromPoint").mockReturnValue([]);
    const first = vi.fn();
    const second = vi.fn();
    initNativeFileDrop({ onFiles: first });
    initNativeFileDrop({ onFiles: second });
    __emitFileDrop(1, 1, ["/a"]);
    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledWith(null, ["/a"], "center");
  });
});

describe("reorderIndex", () => {
  const items = [{ id: "a" }, { id: "b" }, { id: "c" }, { id: "d" }];
  it("moves a later item before an earlier one", () => {
    expect(reorderIndex(items, "d", "b", false)).toBe(1);
  });
  it("moves a later item after an earlier one", () => {
    expect(reorderIndex(items, "d", "b", true)).toBe(2);
  });
  it("moves an earlier item before a later one (post-removal index)", () => {
    expect(reorderIndex(items, "a", "c", false)).toBe(1);
  });
  it("moves an earlier item after a later one (post-removal index)", () => {
    expect(reorderIndex(items, "a", "c", true)).toBe(2);
  });
  it("returns null for the same item or unknown ids", () => {
    expect(reorderIndex(items, "a", "a", false)).toBeNull();
    expect(reorderIndex(items, "x", "a", false)).toBeNull();
    expect(reorderIndex(items, "a", "x", true)).toBeNull();
  });
});
