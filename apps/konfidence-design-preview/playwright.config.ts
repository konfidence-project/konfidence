import { defineConfig } from "@playwright/test";

export default defineConfig({
  testMatch: ["e2e/**/*.test.ts"],
  use: { baseURL: "http://127.0.0.1:4174" },
  webServer: {
    command: "pnpm build && pnpm preview --host 127.0.0.1 --port 4174",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    url: "http://127.0.0.1:4174",
  },
});
