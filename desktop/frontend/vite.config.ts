import { defineConfig } from 'vite'

export default defineConfig({
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(process.env.npm_package_version ?? '0.0.0'),
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-setup.ts',
  },
})
