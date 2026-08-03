import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  server: {
    // Pin the dev-server port so wails.json's frontend:dev:serverUrl always
    // matches. strictPort makes Vite fail loudly instead of drifting to 5174.
    port: 5173,
    strictPort: true,
  },
})
