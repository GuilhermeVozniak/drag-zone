import { describe, expect, it } from "vitest";
import { downloadUrl, latestReleaseUrl, PRODUCT, releaseAssetName } from "./index";

describe("releaseAssetName", () => {
  it("names the macOS universal DMG", () => {
    expect(releaseAssetName("darwin", "universal", "1.2.3")).toBe(
      "dragzone_1.2.3_darwin_universal.dmg",
    );
  });
});

describe("downloadUrl", () => {
  it("builds a tagged release asset URL", () => {
    expect(downloadUrl("darwin", "universal", "1.2.3")).toBe(
      `${PRODUCT.repo}/releases/download/v1.2.3/dragzone_1.2.3_darwin_universal.dmg`,
    );
  });
});

describe("latestReleaseUrl", () => {
  it("points at the latest release page", () => {
    expect(latestReleaseUrl()).toBe(`${PRODUCT.repo}/releases/latest`);
  });
});
