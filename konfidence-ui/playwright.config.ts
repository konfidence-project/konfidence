import { defineConfig } from "@playwright/test";

export default defineConfig({
  testMatch: "**/*.e2e.{ts,js}",
  webServer: {
    command: "pnpm build && pnpm preview",
    port: 4173,
    reuseExistingServer: !process.env.CI,
  },
});
