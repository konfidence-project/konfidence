import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/vite-plugin-svelte').SvelteConfig} */
const config = {
  compilerOptions: {
    // Force runes mode across the design-system package to match `konfidence-ui`.
    runes: true,
  },
  preprocess: vitePreprocess(),
};

export default config;
