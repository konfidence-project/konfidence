import adapter from "@sveltejs/adapter-static";
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
    sveltekit({
      adapter: adapter({ fallback: "index.html" }),
      compilerOptions: {
        // Force runes mode for application code until Svelte 6 makes it the default.
        runes: ({ filename }) => {
          if (filename.split(/[/\\]/).includes("node_modules")) {
            return undefined;
          }

          return true;
        },
      },
      paths: { relative: false },
    }),
  ],
  server: {
    port: 5174,
    strictPort: false,
  },
});
