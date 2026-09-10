import { defineConfig } from "vite";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    {
      configurePreviewServer: (server) => {
        server.middlewares.use((request, _response, next) => {
          const { pathname } = new URL(request.url ?? "/", "http://localhost");
          const filename = pathname.split("/").pop() ?? "";
          if (request.method === "GET" && !filename.includes(".")) {
            request.url = "/";
          }
          next();
        });
      },
      name: "spa-preview-fallback",
    },
    tailwindcss(),
    sveltekit(),
  ],
  server: {
    port: 5174,
    strictPort: false,
  },
});
