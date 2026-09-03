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
      env: {
        node: true,
      },
      files: ["src/tests/**/*.ts"],
      rules: {
        "eslint/max-statements": "off",
        "eslint/no-magic-numbers": "off",
        "import/no-nodejs-modules": "off",
        "unicorn/consistent-function-scoping": "off",
        "unicorn/filename-case": "off",
        "unicorn/no-null": "off",
      },
    },
  ],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
    "eslint/capitalized-comments": ["warn", "always", { ignoreConsecutiveComments: true }],
    "eslint/no-duplicate-imports": ["warn", { allowSeparateTypeImports: true }],
    "eslint/no-ternary": "off",
    "eslint/one-var": "off",
    "import/no-named-export": "off",
    "import/no-namespace": "off",
    "import/no-unassigned-import": ["warn", { allow: ["**/*.css"] }],
    "import/prefer-default-export": "off",
    "no-magic-numbers": ["warn", { ignore: [-1, 0, 1] }],
    "sort-imports": ["warn", { ignoreDeclarationSort: true }],
  },
});
