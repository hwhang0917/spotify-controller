import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  // ../ui/theme.css lives outside this Vite root
  server: { fs: { allow: ['..'] } },
})
