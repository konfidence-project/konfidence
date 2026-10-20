import adapter from "@sveltejs/adapter-static";
import { defineConfig } from "vitest/config";
import { playwright } from "@vitest/browser-playwright";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    {
      configurePreviewServer: (server) => {
        server.middlewares.use((request, _response, next) => {
          const { pathname } = new URL(request.url ?? "/", "http://localhost");
          const filename = pathname.split("/").pop() ?? "";
          if (
            request.method === "GET" &&
            !pathname.startsWith("/api/") &&
            !filename.includes(".")
          ) {
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
    proxy: {
      "/api/v1": process.env.KONFIDENCE_API_URL ?? "http://127.0.0.1:8091",
    },
  },
  test: {
    expect: { requireAssertions: true },
    projects: [
      {
        test: {
          browser: {
            enabled: true,
            instances: [{ browser: "chromium", headless: true }],
            provider: playwright(),
          },
          exclude: ["src/lib/server/**"],
          include: ["src/**/*.svelte.test.{js,ts}"],
          name: "client",
        },
      },

      {
        test: {
          environment: "node",
          exclude: ["src/**/*.svelte.test.{js,ts}"],
          include: ["src/**/*.test.{js,ts}"],
          name: "server",
        },
      },
    ],
  },
});
