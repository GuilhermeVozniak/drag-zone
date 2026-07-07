/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    // Stub the generated Wails modules that a few source files import
    // directly. Test-only: never applied to `vite build`.
    // NOTE: the leading `.*` in each `find` is REQUIRED — rollup-alias (behind
    // Vite's resolve.alias) resolves a regex alias as `id.replace(find, repl)`,
    // so without it the `../../` prefix of the specifier survives and the
    // rewritten id (`../..//abs/stub.ts`) fails to resolve.
    alias: [
      {
        find: /.*wailsjs\/runtime\/runtime$/,
        replacement: path.resolve(__dirname, './src/test/stubs/runtime.ts'),
      },
      {
        find: /.*wailsjs\/go\/main\/App$/,
        replacement: path.resolve(__dirname, './src/test/stubs/App.ts'),
      },
    ],
  },
})
