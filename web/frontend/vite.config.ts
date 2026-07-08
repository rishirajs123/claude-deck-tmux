import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Build straight into web/static so the Go binary's //go:embed static picks it up.
export default defineConfig({
  plugins: [react()],
  build: { outDir: '../static', emptyOutDir: true },
  server: { proxy: { '/api': 'http://localhost:7420' } },
})
