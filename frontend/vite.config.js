import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Build straight into the Go embed directory so the binary ships the UI.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
});
