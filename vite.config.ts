import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/auth': {
        target: 'http://localhost:8026',
        changeOrigin: true
      },
      '/api': {
        target: 'http://localhost:8026',
        changeOrigin: true
      }
    }
  }
});
