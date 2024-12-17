import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import { fileSystemTreePlugin } from './script/generate-webcontainer-fstree'

// https://vite.dev/config/
export default defineConfig({
  base: "./",
  plugins: [
    react(),
    fileSystemTreePlugin("./webcontainer"),
  ],
  server: {
    headers: {
      "Cross-Origin-Embedder-Policy": "require-corp",
      "Cross-Origin-Opener-Policy": "same-origin",
      "Cross-Origin-Resource-Policy": "cross-origin",
    }
  },
})
