import { defineConfig } from "vitest/config";
import { playwright } from "@vitest/browser-playwright";
import { sveltekit } from "@sveltejs/kit/vite";

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
    sveltekit(),
  ],
  preview: {
    proxy: {
      "/api/v1": process.env.KONFIDENCE_API_URL ?? "http://127.0.0.1:8091",
    },
  },
  server: {
    proxy: {
      "/api/v1": process.env.KONFIDENCE_API_URL ?? "http://127.0.0.1:8091",
    },
  },
  test: {
    expect: { requireAssertions: true },
    projects: [
      {
        extends: "./vite.config.ts",
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
        extends: "./vite.config.ts",
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
