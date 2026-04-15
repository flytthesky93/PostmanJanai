import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  // Required for Wails production: absolute "/assets/..." fails to load in embedded WebView
  base: './',
  plugins: [vue()]
})
