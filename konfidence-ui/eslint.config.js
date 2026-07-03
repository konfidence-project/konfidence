import svelte from "eslint-plugin-svelte";
import tsParser from "@typescript-eslint/parser";

const config = [
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

export default config;
