import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [vue(), wails("./bindings")],
  // App de desktop: tudo é lido do disco (sem rede), então um pacote único de ~700 kB não pesa.
  build: { chunkSizeWarningLimit: 1000 },
  // `npm run demo`: troca os serviços Go por dados fictícios (src/demo) para revisar o visual
  // no navegador. Nunca usado no build de produção.
  resolve: mode === "demo"
    ? { alias: [{ find: /^.*\/bindings\/sitposse7\/desktop\/app$/, replacement: fileURLToPath(new URL("./src/demo/app.ts", import.meta.url)) }] }
    : {},
}));
