import { downloadUrl, latestReleaseUrl, type Platform } from "@dragzone/shared";

// Single source of truth for the version the landing page advertises.
// Bump this in lockstep with a desktop release tag.
export const APP_VERSION = "0.3.8";

export type { Platform };
export { downloadUrl, latestReleaseUrl };
