import { defineConfig } from "vitest/config";
import { playwright } from "@vitest/browser-playwright";
import { sveltekit } from "@sveltejs/kit/vite";

export default defineConfig({
  optimizeDeps: {
    include: [
      "@humanspeak/svelte-headless-table/plugins",
      "@ui5/webcomponents-icons/dist/accept.js",
      "@ui5/webcomponents/dist/Table.js",
      "@ui5/webcomponents/dist/TableCell.js",
      "@ui5/webcomponents/dist/TableHeaderCell.js",
      "@ui5/webcomponents/dist/TableHeaderRow.js",
      "@ui5/webcomponents/dist/TableRow.js",
    ],
  },
  plugins: [sveltekit()],
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
          include: ["src/**/*.svelte.{test,spec}.{js,ts}"],
          name: "client",
        },
      },

      {
        extends: "./vite.config.ts",
        test: {
          environment: "node",
          exclude: ["src/**/*.svelte.{test,spec}.{js,ts}"],
          include: ["src/**/*.{test,spec}.{js,ts}"],
          name: "server",
        },
      },
    ],
  },
});
