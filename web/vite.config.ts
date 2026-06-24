import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// Dev: Vite serve o SPA com HMR em :5180 e faz proxy de /v1 -> API Go (:8080),
// então o front roda same-origin sem precisar rebuildar a imagem Docker.
const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    port: 5180,
    strictPort: true,
    proxy: {
      "/v1": { target: apiProxyTarget, changeOrigin: true },
      "/healthz": { target: apiProxyTarget, changeOrigin: true },
    },
  },
  build: {
    outDir: "dist",
  },
});
