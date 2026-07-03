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
      env: {
        node: true,
      },
      files: ["*.config.ts", "*.config.js", "src/lib/server/**/*.ts", "src/routes/api/**/*.ts"],
    },
  ],
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
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
