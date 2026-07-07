import { beforeEach, describe, expect, it, vi } from "vitest";
import { initNativeFileDrop, payloadFromDataTransfer, setUIScale } from "@/lib/dnd";
import { __emitFileDrop, __resetRuntimeStub } from "@/test/stubs/runtime";

function fakeDT(data: Record<string, string>): DataTransfer {
  return { getData: (t: string) => data[t] ?? "" } as unknown as DataTransfer;
}

beforeEach(() => {
  __resetRuntimeStub();
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
    vi.spyOn(document, "elementFromPoint").mockReturnValue(document.getElementById("child"));
    const onFiles = vi.fn();
    initNativeFileDrop({ onFiles });
    __emitFileDrop(100, 200, ["/a.txt"]);
    expect(onFiles).toHaveBeenCalledWith("dropbar", ["/a.txt"]);
  });
  it("un-zooms the cursor coordinates by the UI scale before hit-testing", () => {
    setUIScale(2);
    const spy = vi.spyOn(document, "elementFromPoint").mockReturnValue(null);
    initNativeFileDrop({ onFiles: vi.fn() });
    __emitFileDrop(100, 200, ["/a"]);
    expect(spy).toHaveBeenCalledWith(50, 100);
  });
  it("passes a null drop-id when nothing is under the cursor", () => {
    vi.spyOn(document, "elementFromPoint").mockReturnValue(null);
    const onFiles = vi.fn();
    initNativeFileDrop({ onFiles });
    __emitFileDrop(1, 1, ["/a"]);
    expect(onFiles).toHaveBeenCalledWith(null, ["/a"]);
  });
  it("clears the native-dragging body class on drop", () => {
    document.body.classList.add("native-dragging");
    vi.spyOn(document, "elementFromPoint").mockReturnValue(null);
    initNativeFileDrop({ onFiles: vi.fn() });
    __emitFileDrop(1, 1, []);
    expect(document.body.classList.contains("native-dragging")).toBe(false);
  });
});
