import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 5174,
    proxy: {
      '/admin': {
        target: 'http://127.0.0.1:9013',
        changeOrigin: true
      }
    }
  }
});
