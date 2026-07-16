import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      "/api/": { target: "http://localhost:8080" },
      "/ws": { target: "ws://localhost:8080", ws: true },
      // Mirrors the Netlify proxy that maps this path to the Go API in prod.
      "/sitemap-blocks.xml": { target: "http://localhost:8080" },
    },
  },
});
