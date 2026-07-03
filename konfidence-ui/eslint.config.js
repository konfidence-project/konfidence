import svelte from "eslint-plugin-svelte";
import tsParser from "@typescript-eslint/parser";

export default [
  ...svelte.configs["flat/recommended"],
  {
    files: ["src/**/*.svelte"],
    languageOptions: {
      parserOptions: {
        parser: tsParser,
      },
    },
  },
];
