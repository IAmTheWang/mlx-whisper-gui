import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    // Vite build output writes directly into the Go package's embeddable
    // dist/ dir -- go:embed then has no separate copy step to keep in sync.
    outDir: '../internal/server/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8787',
    },
  },
})
