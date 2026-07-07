import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useActionSpecs, useDropBar, useSettings, useTargets, useTasks } from "@/hooks/useBackend";
import { __fireEvent, __resetBackendMock, __unsub } from "@/lib/__mocks__/backend";
import { backend, type Settings, type Target } from "@/lib/backend";

vi.mock("@/lib/backend");

const target = (id: string): Target => ({ id }) as Target;

beforeEach(() => {
  vi.clearAllMocks();
  __resetBackendMock();
});

describe("useTargets", () => {
  it("loads initial targets then updates on grid:changed", async () => {
    vi.mocked(backend.grid.list).mockResolvedValue([target("a")] as never);
    const { result } = renderHook(() => useTargets());
    await waitFor(() => expect(result.current).toEqual([target("a")]));
    act(() => __fireEvent("grid:changed", [target("a"), target("b")]));
    expect(result.current).toEqual([target("a"), target("b")]);
  });
  it("unsubscribes from grid:changed on unmount", async () => {
    vi.mocked(backend.grid.list).mockResolvedValue([] as never);
    const { unmount } = renderHook(() => useTargets());
    await waitFor(() => expect(__unsub["grid:changed"]).toBeDefined());
    unmount();
    expect(__unsub["grid:changed"]).toHaveBeenCalled();
  });
});

describe("useTasks / useDropBar / useActionSpecs coerce null to []", () => {
  it("useTasks maps a null binding result to an empty array", async () => {
    vi.mocked(backend.tasks.list).mockResolvedValue(null as never);
    const { result } = renderHook(() => useTasks());
    await waitFor(() => expect(backend.tasks.list).toHaveBeenCalled());
    expect(result.current).toEqual([]);
  });
  it("useDropBar updates on dropbar:changed", async () => {
    vi.mocked(backend.dropBar.list).mockResolvedValue([] as never);
    const { result } = renderHook(() => useDropBar());
    await waitFor(() => expect(backend.dropBar.list).toHaveBeenCalled());
    act(() => __fireEvent("dropbar:changed", [{ id: "x" }]));
    expect(result.current).toEqual([{ id: "x" }]);
  });
  it("useActionSpecs updates on specs:changed", async () => {
    vi.mocked(backend.actions.specs).mockResolvedValue([] as never);
    const { result } = renderHook(() => useActionSpecs());
    await waitFor(() => expect(backend.actions.specs).toHaveBeenCalled());
    act(() => __fireEvent("specs:changed", [{ id: "zip" }]));
    expect(result.current).toEqual([{ id: "zip" }]);
  });
});

describe("useSettings", () => {
  // Relies on the module-level settings singleton being null at first use;
  // this is the only test in the suite that consumes it.
  it("loads once, then update() persists and republishes to every consumer", async () => {
    vi.mocked(backend.settings.get).mockResolvedValue({ gridSize: 40 } as Settings as never);
    const a = renderHook(() => useSettings());
    await waitFor(() => expect(a.result.current[0]).toEqual({ gridSize: 40 }));

    // a second consumer sees the already-loaded value with no extra fetch
    vi.mocked(backend.settings.get).mockClear();
    const b = renderHook(() => useSettings());
    expect(b.result.current[0]).toEqual({ gridSize: 40 });
    expect(backend.settings.get).not.toHaveBeenCalled();

    await act(async () => {
      await a.result.current[1]({ gridSize: 80 } as Settings);
    });
    expect(backend.settings.set).toHaveBeenCalledWith({ gridSize: 80 });
    expect(a.result.current[0]).toEqual({ gridSize: 80 });
    expect(b.result.current[0]).toEqual({ gridSize: 80 });
  });
});
