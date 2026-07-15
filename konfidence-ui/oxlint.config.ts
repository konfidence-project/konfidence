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
        // Svelte 5 `bind:this` requires `let`; the compiler assigns internally even if user code never reassigns, so `const` is rejected.
        "prefer-const": "off",
        // Svelte components conventionally use PascalCase filenames in this project.
        "unicorn/filename-case": "off",
      },
    },
    {
      files: ["src/app.d.ts", "src/routes/**/+*.ts", "src/routes/**/+*.js"],
      rules: {
        // SvelteKit route modules and app declarations require named exports/module markers.
        "import/exports-last": "off",
        "import/group-exports": "off",
        "import/no-named-export": "off",
        "import/prefer-default-export": "off",
      },
    },
    {
      files: ["src/**/*.remote.ts"],
      rules: {
        // SvelteKit remote modules forbid `export default` — must use named exports.
        "import/prefer-default-export": "off",
      },
    },
    {
      env: {
        node: true,
      },
      files: ["*.config.ts", "*.config.js", "src/lib/server/**/*.ts", "src/routes/api/**/*.ts"],
    },
    {
      env: {
        node: true,
      },
      files: ["tests/auth/**/*.ts"],
      rules: {
        // Integration setup is intentionally procedural and uses Node process/container APIs.
        "eslint/max-params": "off",
        "eslint/max-statements": "off",
        "eslint/no-await-in-loop": "off",
        "import/no-nodejs-modules": "off",
      },
    },
  ],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
    // Conflicts with `no-duplicate-imports` and `typescript/consistent-type-imports`:
    // Those two already force inline `type` specifiers when importing both values and types from one module.
    "import/consistent-type-specifier-style": "off",

    // The app uses named exports intentionally for shared utilities, stores, and route-adjacent modules.
    "import/no-named-export": "off",

    "import/no-unassigned-import": [
      "warn",
      {
        allow: ["@ui5/**", "**/*.css"],
      },
    ],

    // Allow trivial sentinel values (array indices, length checks, offsets) inline.
    // Real magic numbers (e.g. 60_000, 500) still require named constants.
    "no-magic-numbers": [
      "warn",
      {
        ignore: [0, 1, -1],
      },
    ],
  },
});
