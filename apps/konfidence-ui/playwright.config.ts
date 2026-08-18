import { defineConfig } from "@playwright/test";

export default defineConfig({
  testMatch: ["src/**/*.test-e2e.{ts,js}", "e2e/**/*.test.{ts,js}"],
  use: {
    baseURL: "http://127.0.0.1:4173",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "pnpm build && HOST=127.0.0.1 PORT=4173 pnpm start",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    url: "http://127.0.0.1:4173",
  },
});
