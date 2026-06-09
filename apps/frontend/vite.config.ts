import { defineConfig } from "vite";

export default defineConfig({
  build: {
    outDir: "../../substrate/go/frontend/site",
    emptyOutDir: true,
    sourcemap: false,
  },
});
