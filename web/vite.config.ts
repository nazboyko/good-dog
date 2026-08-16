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
  test: {
    // most tests are pure and need no dom. The ones that click a button
    // ask for jsdom with a docblock, so the rest stay fast
    environment: "node",
    environmentMatchGlobs: [["**/*.dom.test.tsx", "jsdom"]],
  },
});
