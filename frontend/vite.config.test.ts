import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import path from "path";

// Test-specific Vite config: replaces @wailsio/runtime with a mock
// so the frontend can run standalone without the Go backend.
export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      "@wailsio/runtime": path.resolve(__dirname, "tests/mocks/wails-runtime.ts"),
    },
  },
});
