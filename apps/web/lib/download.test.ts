import { describe, expect, it } from "vitest";
import { APP_VERSION, downloadUrl } from "./download";

describe("landing download", () => {
  it("advertises a well-formed macOS DMG url for the pinned version", () => {
    const url = downloadUrl("darwin", "universal", APP_VERSION);
    expect(url).toMatch(
      /\/releases\/download\/v\d+\.\d+\.\d+\/dragzone_\d+\.\d+\.\d+_darwin_universal\.dmg$/,
    );
  });
});
