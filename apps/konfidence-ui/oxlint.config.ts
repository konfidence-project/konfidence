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
  ignorePatterns: ["src/lib/konfidence-api/schema.d.ts"],
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
    {
      env: {
        node: true,
      },
      files: ["src/**/*.test.ts", "src/**/*.svelte.test.ts"],
      rules: {
        "eslint/max-statements": "off",
        "eslint/no-magic-numbers": "off",
        "import/no-nodejs-modules": "off",
        "unicorn/consistent-function-scoping": "off",
        "unicorn/filename-case": "off",
        "unicorn/no-null": "off",
      },
    },
    {
      files: ["src/lib/theme/**/*.ts", "src/lib/theme/**/*.svelte.ts"],
      rules: {
        // The theme resolver + store bridge the DOM, localStorage,
        // history, and matchMedia — a handful of statements is
        // unavoidable and splitting them into helpers would obscure
        // the linear control flow more than it clarifies.
        "eslint/max-statements": "off",
        // `history.replaceState(null, ...)` is required by the API
        // (undefined would serialise as the string "undefined").
        "unicorn/no-null": "off",
      },
    },
  ],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
    // Multi-line `//` comments frequently start with lowercase continuation
    // lines; ignore consecutive comments so only the first line is checked.
    "eslint/capitalized-comments": ["warn", "always", { ignoreConsecutiveComments: true }],
    // Openapi-fetch exposes HTTP verbs as uppercase methods (`client.GET`,
    // `client.POST`, …); they are not constructors.
    "eslint/new-cap": ["warn", { capIsNewExceptions: ["DELETE", "GET", "PATCH", "POST", "PUT"] }],
    // Allow `import type { Foo }` and `import { bar }` from the same module
    // on separate lines; this is our convention (see `import/consistent-type-specifier-style`).
    "eslint/no-duplicate-imports": ["warn", { allowSeparateTypeImports: true }],
    // Ternaries are idiomatic in TypeScript/Svelte for concise conditional
    // values; forbidding them adds no value.
    "eslint/no-ternary": "off",
    // Splitting `const` declarations onto their own lines is our house style;
    // combining them into a single `const a = …, b = …;` hurts readability.
    "eslint/one-var": "off",
    "import/no-named-export": "off",
    "import/no-namespace": "off",
    // Global stylesheets are imported for their side effects.
    "import/no-unassigned-import": ["warn", { allow: ["**/*.css"] }],
    "import/prefer-default-export": "off",
    "no-magic-numbers": ["warn", { ignore: [-1, 0, 1, 200, 302, 400, 401, 403, 404, 500] }],
    "sort-imports": ["warn", { ignoreDeclarationSort: true }],
  },
});
