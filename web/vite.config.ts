import { defineConfig, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'

function inlineTouchIcon(): Plugin {
  return {
    name: 'inline-touch-icon',
    apply: 'build',
    transformIndexHtml(html) {
      const icon = readFileSync(new URL('./public/apple-touch-icon.png', import.meta.url)).toString('base64')
      return html.replace('href="/apple-touch-icon.png"', `href="data:image/png;base64,${icon}"`)
    },
  }
}

export default defineConfig({
  plugins: [react(), inlineTouchIcon()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: '../pkg/server/api/dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-xyflow': ['@xyflow/react'],
          'vendor-xterm': ['@xterm/xterm', '@xterm/addon-fit'],
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
