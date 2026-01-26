import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { svgPlugin } from 'vite-plugin-fast-react-svg';

export default defineConfig({
  plugins: [
    svgPlugin(),
    react(),
  ],
  css: {
    preprocessorOptions: {
      less: {
        math: 'always',
        javascriptEnabled: true,
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/setupTests.ts',
  },
});
