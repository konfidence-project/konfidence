import adapter from "@sveltejs/adapter-static";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    adapter: adapter({
      pages: "../../internal/kden/auth/pages/generated",
      assets: ".svelte-kit/static-assets",
    }),
    inlineStyleThreshold: 1_000_000,
  },
};

export default config;
