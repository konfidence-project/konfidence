import svelte from "eslint-plugin-svelte";
import tsParser from "@typescript-eslint/parser";

const config = [
  ...svelte.configs["flat/recommended"],
  {
    files: ["src/**/*.svelte"],
    languageOptions: {
      parserOptions: {
        extraFileExtensions: [".svelte"],
        parser: tsParser,
        projectService: true,
      },
    },
  },
];

export default config;
