import { defineConfig } from "@playwright/test";

export default defineConfig({
  testMatch: "**/*.e2e.{ts,js}",
  use: {
    baseURL: "http://127.0.0.1:4174",
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: "pnpm dev:mock-api",
      env: { KONFIDENCE_MOCK_API_PORT: "18091" },
      gracefulShutdown: { signal: "SIGTERM", timeout: 1000 },
      name: "Mock API",
      reuseExistingServer: !process.env.CI,
      url: "http://127.0.0.1:18091/health",
    },
    {
      command: "pnpm build && pnpm preview --host 127.0.0.1 --port 4174",
      env: { KONFIDENCE_API_URL: "http://127.0.0.1:18091" },
      gracefulShutdown: { signal: "SIGTERM", timeout: 1000 },
      name: "SvelteKit",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      url: "http://127.0.0.1:4174/login",
    },
  ],
});
