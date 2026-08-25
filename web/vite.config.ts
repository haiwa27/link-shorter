import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Im Entwicklungsbetrieb gehen API-Aufrufe an das Go-Backend. /healthz
    // gehoert dazu, weil das Zustandsschild im Kopf es abfragt.
    proxy: {
      "/api": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
  },
});
