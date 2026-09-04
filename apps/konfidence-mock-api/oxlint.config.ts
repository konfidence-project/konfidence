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
    node: true,
  },
  plugins: ["typescript", "unicorn", "oxc", "eslint", "import"],
  rules: {
    "eslint/init-declarations": "off",
    "eslint/max-statements": "off",
    "eslint/no-ternary": "off",
    "import/consistent-type-specifier-style": "off",
    "import/no-named-export": "off",
    "import/no-nodejs-modules": "off",
    "import/prefer-default-export": "off",
    "no-magic-numbers": ["warn", { ignore: [-1, 0, 1, 200, 302, 400, 401, 403, 404, 405, 500] }],
    "sort-imports": "off",
  },
});
