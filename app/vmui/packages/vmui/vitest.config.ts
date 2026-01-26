import { defineConfig } from "vitest/config";
import preact from "@preact/preset-vite";

export default defineConfig({
  plugins: [preact({ reactAliasesEnabled: false })],
  resolve: {
    alias: {
      "react-dom/test-utils": "preact/test-utils",
      "react-dom": "preact/compat",
      "react/jsx-runtime": "preact/jsx-runtime",
      "react": `${import.meta.dirname}/src/compat/react.ts`,
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: "./src/setupTests.ts",
  },
});
