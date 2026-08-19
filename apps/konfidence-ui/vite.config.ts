import { defineConfig } from "vitest/config";
import { playwright } from "@vitest/browser-playwright";
import { sveltekit } from "@sveltejs/kit/vite";

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      "/api/v1": "http://127.0.0.1:8090",
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
