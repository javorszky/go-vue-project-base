import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { defineConfig, type Plugin } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// emptyOutDir wipes the tracked .gitkeep from internal/ui/dist on every
// build, which would leave the working tree dirty. Recreate it after the
// bundle is written — it exists so `go:embed all:dist` compiles on a fresh
// clone, before any frontend build has run.
const restoreGitkeep = (): Plugin => ({
  name: 'restore-gitkeep',
  closeBundle() {
    writeFileSync(fileURLToPath(new URL('../internal/ui/dist/.gitkeep', import.meta.url)), '')
  },
})

export default defineConfig({
  plugins: [vue(), tailwindcss(), restoreGitkeep()],
  build: {
    // Output goes into the Go embed package, not frontend/dist.
    outDir: '../internal/ui/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    passWithNoTests: true,
    coverage: {
      provider: 'v8',
      reporter: ['lcov'],
      exclude: ['node_modules/', 'dist/', '**/*.config.*'],
    },
  },
})
