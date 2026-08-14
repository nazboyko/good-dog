import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// /events must stay uncompressed and unbuffered or SSE dies, see docs/tech-stack.md
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/events": "http://localhost:8080",
      "/api": "http://localhost:8080",
    },
  },
});
