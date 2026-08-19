import { defineConfig } from "@playwright/test";

export default defineConfig({
  testMatch: ["src/**/*.test-e2e.{ts,js}", "e2e/**/*.test.{ts,js}"],
  use: {
    baseURL: "http://127.0.0.1:4173",
    trace: "retain-on-failure",
  },
  webServer: {
    command:
      "pnpm build && KUBECONFIG=../../internal/api/server/testdata/kubeconfig.yaml go run ../../cmd/api --addr=127.0.0.1:4173 --ui-asset-path=build --log-level=error --oidc-enabled=false",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    url: "http://127.0.0.1:4173",
  },
});
