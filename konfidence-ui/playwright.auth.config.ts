import { defineConfig } from "@playwright/test";

export default defineConfig({
  fullyParallel: false,
  globalSetup: "./tests/auth/global-setup.ts",
  testMatch: "tests/auth/**/*.e2e.ts",
  use: {
    baseURL: "http://127.0.0.1:4173",
  },
  workers: 1,
});
