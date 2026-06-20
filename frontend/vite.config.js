import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// 前端构建产物直接输出到 Go 服务的 embed 目录，单二进制同源部署。
// 开发时通过 proxy 把 /api 和 /healthz 转发到本地后端（默认 :8080）。
const backend = process.env.MODELISTER_BACKEND || "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../internal/webui/dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": { target: backend, changeOrigin: true },
      "/healthz": { target: backend, changeOrigin: true },
    },
  },
});
