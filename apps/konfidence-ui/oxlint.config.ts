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
    builtin: true,
  },
  overrides: [
    {
      files: ["src/**/*.svelte", "src/**/*.svelte.ts"],
      globals: {
        $derived: "readonly",
        $effect: "readonly",
        $inspect: "readonly",
        $props: "readonly",
        $state: "readonly",
      },
      rules: {
        "eslint/capitalized-comments": "off",
        "eslint/no-implicit-coercion": "off",
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
    "import/no-named-export": "off",
    "import/no-namespace": "off",
    "import/prefer-default-export": "off",
    "no-magic-numbers": ["warn", { ignore: [-1, 0, 1, 200, 302, 400, 401, 403, 404, 500] }],
    "sort-imports": ["warn", { ignoreDeclarationSort: true }],
  },
});
