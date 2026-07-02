import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  base: '/data-readiness-tabular/',
  plugins: [react()],
  server: {
    proxy: {
      '/data-readiness-tabular/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/data-readiness-tabular/, ''),
      },
    },
  },
})
