import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "src/__tests__",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  updateSnapshots: "none",
  snapshotPathTemplate:
    "{testDir}/__screenshots__/{testFilePath}/{arg}-{projectName}-{platform}{ext}",
  use: {
    baseURL: "http://cli.invalid",
    browserName: "chromium",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium" }],
});
