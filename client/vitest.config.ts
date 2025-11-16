import path from "node:path";
import solidPlugin from "vite-plugin-solid";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [solidPlugin()],
  test: {
    environment: "jsdom",
    setupFiles: [path.resolve(__dirname, "./src/test/setup.ts")],
    deps: {
      optimizer: {
        web: {
          include: ["solid-js"],
        },
      },
    },
    globals: true,
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
});
