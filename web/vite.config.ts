import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// Dev only: proxy API calls to a running guest server (start it from the admin UI).
const API_PROXY = process.env.VITE_API_PROXY ?? 'http://127.0.0.1:5555'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: { proxy: { '/api': API_PROXY } },
})
