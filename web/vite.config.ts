import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// Dev: Vite serve o SPA com HMR em :5173 e faz proxy de /v1 -> API Go (:8080),
// então o front roda same-origin sem precisar rebuildar a imagem Docker.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    port: 5180,
    strictPort: true,
    proxy: {
      "/v1": { target: "http://localhost:8080", changeOrigin: true },
      "/healthz": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
  build: {
    outDir: "dist",
  },
});
