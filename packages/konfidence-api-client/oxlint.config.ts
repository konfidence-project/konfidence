import { defineConfig } from "oxlint";

export default defineConfig({
  $schema: "./node_modules/oxlint/configuration_schema.json",
  categories: {
    correctness: "error",
    nursery: "warn",
    perf: "warn",
    suspicious: "warn",
  },
  ignorePatterns: ["src/schema.d.ts"],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
    "import/no-named-export": "off",
    "import/prefer-default-export": "off",
  },
});
