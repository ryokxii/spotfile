/// <reference types="vitest" />
import {fileURLToPath} from 'node:url'
import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      // Lets CSS url() reference bundled font files from the @fontsource
      // packages; Vite 3 does not resolve bare package paths inside CSS.
      '@fontsource': fileURLToPath(new URL('./node_modules/@fontsource', import.meta.url)),
    },
  },
  server: {
    // Pin the dev-server port so wails.json's frontend:dev:serverUrl always
    // matches. strictPort makes Vite fail loudly instead of drifting to 5174.
    port: 5173,
    strictPort: true,
  },
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'jsdom',
  },
})
