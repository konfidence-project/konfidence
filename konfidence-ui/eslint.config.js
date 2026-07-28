import svelte from "eslint-plugin-svelte";
import tsParser from "@typescript-eslint/parser";

const config = [
  { ignores: ["src/lib/components/ui/**"] },
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
  {
    files: ["src/lib/components/AppNavigation.svelte"],
    rules: { "svelte/no-navigation-without-resolve": "off" },
  },
];

export default config;
