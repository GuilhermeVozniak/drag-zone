import { describe, expect, it } from "vitest"
import type { ActionSpec } from "@/lib/backend"
import { clickBehavior } from "./clickBehavior"

const spec = (over: Partial<ActionSpec>): ActionSpec =>
  ({
    id: "x",
    name: "X",
    description: "",
    icon: "",
    category: "",
    events: [],
    accepts: [],
    options: [],
    multi: false,
    ...over,
  }) as ActionSpec

describe("clickBehavior", () => {
  it("runs the click handler when the action declares a clicked event", () => {
    expect(clickBehavior(spec({ events: ["dragged", "clicked"] }))).toBe("run")
  })

  it("opens config for a drag-only action that has options", () => {
    expect(
      clickBehavior(
        spec({
          events: ["dragged"],
          options: [{ key: "path", label: "Path", type: "folder" }],
        })
      )
    ).toBe("config")
  })

  it("does nothing for a drag-only action with no options", () => {
    expect(clickBehavior(spec({ events: ["dragged"] }))).toBe("none")
    expect(clickBehavior(spec({ events: ["dragged"], options: [] }))).toBe("none")
  })

  it("defers to the backend (run) when the spec is missing", () => {
    expect(clickBehavior(undefined)).toBe("run")
  })
})
