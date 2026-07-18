import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  webServer: {
    // Build against the mock backend (e2e/mock/backend.ts, aliased in for
    // wailsjs in vite.config.ts) rather than `wails build`, so this runs
    // without generated bindings — including on ubuntu CI.
    command: "bunx vite build --mode e2e && bunx vite preview --port 4173",
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
  },
  use: { baseURL: "http://localhost:4173" },
});
