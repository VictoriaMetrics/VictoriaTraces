import { defineConfig, ProxyOptions } from "vite";
import preact from "@preact/preset-vite";
import dynamicIndexHtmlPlugin from "./config/plugins/dynamicIndexHtml.ts";

const getProxy = (): Record<string, ProxyOptions> | undefined => {
  const playground = process.env.PLAYGROUND;

  switch (playground) {
    case "TRACES": {
      return {
        "^(/select/.*|/flags)": {
          target: "https://play-vtraces.victoriametrics.com",
          changeOrigin: true,
          configure: (proxy) => {
            proxy.on("proxyReq", (proxyReq) => {
              proxyReq.removeHeader("AccountID");
              proxyReq.removeHeader("ProjectID");
            });

            proxy.on("error", (err) => {
              console.error("[proxy error]", err.message);
            });
          }
        }
      };
    }
    default: {
      return undefined;
    }
  }
};

export default defineConfig(({ mode }) => {
  return {
    base: "",
    plugins: [
      preact(),
      dynamicIndexHtmlPlugin({ mode })
    ],
    assetsInclude: ["**/*.md"],
    server: {
      open: true,
      port: 3000,
      proxy: getProxy(),
    },
    resolve: {
      alias: {
        "src": `${import.meta.dirname}/src`,
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
          },
          assetFileNames: (assetInfo) => {
            if (assetInfo.names.includes("favicon.svg")) {
              return "assets/favicon.svg";
            }

            return "assets/[name]-[hash][extname]";
          },
        }
      }
    },
  };
});



