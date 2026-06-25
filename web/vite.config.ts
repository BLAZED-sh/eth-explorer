import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      "/api/": { target: "http://localhost:8060" },
      "/ws": { target: "ws://localhost:8060", ws: true },
    },
  },
});
