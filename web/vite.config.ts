import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-setup.ts',
    testTimeout: 20000,
  },
})
