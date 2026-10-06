import { defineConfig } from "oxlint";

export default defineConfig({
  categories: { correctness: "error", suspicious: "warn" },
  env: { browser: true, node: true },
  ignorePatterns: ["test-results/**", "playwright-report/**"],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
});
