import { defineConfig } from "vitest/config";
import { playwright } from "@vitest/browser-playwright";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  test: {
    expect: { requireAssertions: true },
    projects: [
      {
        test: {
          browser: {
            commands: {
              resetPointer: async ({ page }) => {
                await page.mouse.move(0, 0);
              },
            },
            enabled: true,
            instances: [{ browser: "chromium", headless: true }],
            provider: playwright(),
            ui: false,
          },
          include: ["src/**/*.svelte.test.{js,ts}"],
          name: "client",
          setupFiles: ["./src/test/browser-setup.ts"],
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
