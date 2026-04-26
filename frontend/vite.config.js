import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  // Required for Wails production: absolute "/assets/..." fails to load in embedded WebView
  base: './',
  plugins: [vue()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined
          if (id.includes('@codemirror') || id.includes('/codemirror/')) {
            return 'vendor-codemirror'
          }
          if (id.includes('/vue/') || id.includes('@vue')) {
            return 'vendor-vue'
          }
          if (id.includes('xml-formatter')) {
            return 'vendor-formatters'
          }
          return 'vendor'
        }
      }
    }
  }
})
