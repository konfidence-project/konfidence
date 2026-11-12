import { defineConfig } from "oxlint";

export default defineConfig({
  $schema: "./node_modules/oxlint/configuration_schema.json",
  categories: {
    correctness: "error",
    nursery: "warn",
    pedantic: "off",
    perf: "warn",
    restriction: "off",
    style: "warn",
    suspicious: "warn",
  },
  env: {
    browser: true,
    builtin: true,
  },
  overrides: [
    {
      env: { node: true },
      files: ["src/lib/playground/*.test.ts"],
      rules: { "import/no-nodejs-modules": "off" },
    },
    {
      files: ["src/lib/playground/compiler.worker.ts", "src/lib/playground/compiler-client.ts"],
      // Worker.postMessage has no Window-style targetOrigin argument.
      rules: { "unicorn/require-post-message-target-origin": "off" },
    },
    {
      files: ["e2e/**/*.ts"],
      rules: { "eslint/max-statements": "off" },
    },
    {
      files: ["src/**/*.svelte", "src/**/*.svelte.ts"],
      globals: {
        $bindable: "readonly",
        $derived: "readonly",
        $effect: "readonly",
        $inspect: "readonly",
        $props: "readonly",
        $state: "readonly",
      },
      rules: {
        "eslint/capitalized-comments": "off",
        "eslint/max-lines": "off",
        "eslint/max-lines-per-function": "off",
        "eslint/no-implicit-coercion": "off",
        "eslint/no-magic-numbers": "off",
        "prefer-const": "off",
        "unicorn/filename-case": "off",
      },
    },
    {
      env: {
        node: true,
      },
      files: ["*.config.ts", "*.config.js"],
    },
    {
      files: ["src/app.d.ts"],
      rules: {
        "eslint/capitalized-comments": "off",
        "unicorn/require-module-specifiers": "off",
      },
    },
  ],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
    "eslint/capitalized-comments": ["warn", "always", { ignoreConsecutiveComments: true }],
    "eslint/new-cap": ["warn", { capIsNewExceptions: ["DELETE", "GET", "PATCH", "POST", "PUT"] }],
    "eslint/no-duplicate-imports": ["warn", { allowSeparateTypeImports: true }],
    "eslint/no-ternary": "off",
    "eslint/one-var": "off",
    "import/no-named-export": "off",
    "import/no-namespace": "off",
    "import/no-unassigned-import": ["warn", { allow: ["**/*.css", "@ui5/**"] }],
    "import/prefer-default-export": "off",
    "no-magic-numbers": "off",
    "sort-imports": ["warn", { ignoreDeclarationSort: true }],
  },
});
