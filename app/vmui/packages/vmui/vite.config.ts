import * as path from "path";

import { defineConfig, ProxyOptions } from "vite";
import react from "@vitejs/plugin-react";
import dynamicIndexHtmlPlugin from "./config/plugins/dynamicIndexHtml";

export default defineConfig(({ mode }) => {
  return {
    base: "",
    plugins: [
      react(),
      dynamicIndexHtmlPlugin({ mode })
    ],
    assetsInclude: ["**/*.md"],
    server: {
      host: "0.0.0.0",
      open: true,
      port: 3000,
      allowedHosts: ['local.vtraces.test'],
    },
    resolve: {
      alias: {
        "src": path.resolve(__dirname, "src"),
      },
    },
    build: {
      outDir: "./build",
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes("node_modules")) {
              return "vendor";
            }
          }
        }
      }
    },
    define: {
      __REACT_APP_GA_DEBUG__: JSON.stringify(process.env.REACT_APP_GA_DEBUG || ''),
      __REACT_APP_VSN_STATE__: JSON.stringify(process.env.REACT_APP_VSN_STATE || ''),
      __APP_ENVIRONMENT__: JSON.stringify(process.env.NODE_ENV || 'development'),
    },
  };
});



