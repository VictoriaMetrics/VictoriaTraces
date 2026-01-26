import { defineConfig, ProxyOptions } from "vite";
import preact from "@preact/preset-vite";
import dynamicIndexHtmlPlugin from "./config/plugins/dynamicIndexHtml.ts";

export default defineConfig(({ mode }) => {
  return {
    base: "",
    plugins: [
      preact({ reactAliasesEnabled: false }),
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
        "src": `${import.meta.dirname}/src`,
        "react-dom/test-utils": "preact/test-utils",
        "react-dom": "preact/compat",
        "react/jsx-runtime": "preact/jsx-runtime",
        "react": `${import.meta.dirname}/src/compat/react.ts`,
      },
      preserveSymlinks: true,
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



