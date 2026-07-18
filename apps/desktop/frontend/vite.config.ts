/// <reference types="vitest/config" />

import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { configDefaults } from "vitest/config";

export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      { find: "@", replacement: path.resolve(__dirname, "./src") },
      // e2e mode (`vite build --mode e2e`, driven by playwright.config.ts):
      // no wailsjs bindings exist (this isn't a `wails build`), so point the
      // same specifiers src/lib/backend.ts and src/lib/dnd.ts import at the
      // stateful mock in e2e/mock/backend.ts. Same regex-alias mechanism as
      // `test.alias` below — the leading `.*` is REQUIRED for the same
      // reason: rollup-alias resolves a regex alias as `id.replace(find,
      // repl)`, so without it the `../../` prefix of the specifier survives
      // and the rewritten id fails to resolve.
      ...(mode === "e2e"
        ? [
            {
              find: /.*wailsjs\/runtime\/runtime$/,
              replacement: path.resolve(__dirname, "./e2e/mock/backend.ts"),
            },
            {
              find: /.*wailsjs\/go\/main\/App$/,
              replacement: path.resolve(__dirname, "./e2e/mock/backend.ts"),
            },
          ]
        : []),
    ],
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    // The e2e/ dir holds Playwright specs (run via `bun run e2e`, not
    // vitest) plus the mock backend they build against; vitest's default
    // `include` would otherwise pick up e2e/*.spec.ts too.
    exclude: [...configDefaults.exclude, "e2e/**"],
    // Stub the generated Wails modules that a few source files import
    // directly. Test-only: never applied to `vite build`.
    // NOTE: the leading `.*` in each `find` is REQUIRED — rollup-alias (behind
    // Vite's resolve.alias) resolves a regex alias as `id.replace(find, repl)`,
    // so without it the `../../` prefix of the specifier survives and the
    // rewritten id (`../..//abs/stub.ts`) fails to resolve.
    alias: [
      {
        find: /.*wailsjs\/runtime\/runtime$/,
        replacement: path.resolve(__dirname, "./src/test/stubs/runtime.ts"),
      },
      {
        find: /.*wailsjs\/go\/main\/App$/,
        replacement: path.resolve(__dirname, "./src/test/stubs/App.ts"),
      },
    ],
  },
}));
