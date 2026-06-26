import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
      '/livekit': {
        target: 'ws://localhost:7880',
        ws: true,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/livekit/, '') || '/',
      },
    },
  },
})
