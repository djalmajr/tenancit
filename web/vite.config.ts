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
    manifest: true,
    outDir: "dist",
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return undefined;
          if (/[\\/]node_modules[\\/](react|react-dom|scheduler)[\\/]/.test(id)) {
            return "react-vendor";
          }
          if (id.includes("/node_modules/@tanstack/")) return "tanstack-vendor";
          if (id.includes("/node_modules/@base-ui/")) return "base-ui-vendor";
          if (id.includes("/node_modules/lucide-react/")) return "icons-vendor";
          if (id.includes("/node_modules/date-fns/")) return "date-vendor";
          if (id.includes("/node_modules/cmdk/")) return "command-vendor";
          return "vendor";
        },
      },
    },
  },
});
