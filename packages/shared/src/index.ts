export const PRODUCT = {
  name: "dragzone",
  displayName: "DragZone",
  repo: "https://github.com/GuilhermeVozniak/drag-zone",
  site: "https://drag-zone.vozniak.dev",
} as const;

// DragZone ships a single macOS universal build; the platform/arch parameters
// keep the API shaped like a multi-platform product so the landing page and
// release workflow agree on one asset-naming contract.
export type Platform = "darwin";
export type Arch = "universal";

const EXTENSIONS: Record<Platform, string> = {
  darwin: "dmg",
};

export function releaseAssetName(platform: Platform, arch: Arch, version: string): string {
  return `${PRODUCT.name}_${version}_${platform}_${arch}.${EXTENSIONS[platform]}`;
}

export function downloadUrl(platform: Platform, arch: Arch, version: string): string {
  return `${PRODUCT.repo}/releases/download/v${version}/${releaseAssetName(platform, arch, version)}`;
}

export function latestReleaseUrl(): string {
  return `${PRODUCT.repo}/releases/latest`;
}
