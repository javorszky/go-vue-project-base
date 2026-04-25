import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
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
