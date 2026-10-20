import adapter from "@sveltejs/adapter-static";
import { defineConfig } from "vite";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit({
      adapter: adapter({
        pages: "../../internal/kden/auth/pages/generated",
        assets: ".svelte-kit/static-assets",
      }),
      inlineStyleThreshold: 1_000_000,
    }),
  ],
});
