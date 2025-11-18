import path from "node:path";
import { defineConfig } from "vite";
import solidPlugin from "vite-plugin-solid";

export default defineConfig({
  envDir: "../",
  plugins: [solidPlugin()],
  server: {
    port: 3021,
    host: "0.0.0.0",
    hmr: {
      protocol: "ws",
      host: "localhost",
      port: 3021,
    },
    watch: {
      usePolling: false,
      ignored: ["**/.git/**", "**/node_modules/**", "**/dist/**"],
    },
  },
  test: {
    globals: false,
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
    deps: {
      optimizer: {
        web: {
          include: ["solid-js", "@solidjs/testing-library"],
        },
      },
    },
    server: {
      deps: {
        inline: [/@solidjs\/router/],
      },
    },
  },
  resolve: {
    alias: {
      "@styles": path.resolve(__dirname, "./src/styles"),
      "@components": path.resolve(__dirname, "./src/components"),
      "@layout": path.resolve(__dirname, "./src/components/layout/"),
      "@pages": path.resolve(__dirname, "./src/pages/"),
      "@hooks": path.resolve(__dirname, "./src/hooks/"),
      "@services": path.resolve(__dirname, "./src/services/"),
      "@context": path.resolve(__dirname, "./src/context/"),
      "@models": path.resolve(__dirname, "./src/types/"),
      "@constants": path.resolve(__dirname, "./src/constants/"),
      "@utils": path.resolve(__dirname, "./src/utils/"),
    },
    conditions: ["development", "browser"],
    extensions: [".mjs", ".js", ".mts", ".ts", ".jsx", ".tsx", ".json"],
  },
  build: {
    target: "esnext",
    cssCodeSplit: true,
    modulePreload: {
      polyfill: true,
    },
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["solid-js", "@solidjs/router"],
          query: ["@tanstack/solid-query"],
          oidc: ["oidc-client-ts"],
        },
        assetFileNames: (assetInfo) => {
          const info = assetInfo.name.split(".");
          const ext = info[info.length - 1];
          if (/png|jpe?g|svg|gif|tiff|bmp|ico/i.test(ext)) {
            return "images/[name]-[hash][extname]";
          }
          if (/woff|woff2|eot|ttf|otf/i.test(ext)) {
            return "fonts/[name]-[hash][extname]";
          }
          return "assets/[name]-[hash][extname]";
        },
        chunkFileNames: "assets/[name]-[hash].js",
        entryFileNames: "assets/[name]-[hash].js",
      },
    },
    minify: "esbuild",
    sourcemap: false,
    chunkSizeWarningLimit: 1000,
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `
          @use "@styles/variables" as *;
          @use "@styles/mixins" as *;
          @use "@styles/colors" as *;
        `,
      },
    },
  },
});
