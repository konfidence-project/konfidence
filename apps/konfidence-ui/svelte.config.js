import adapter from "@sveltejs/adapter-static";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  compilerOptions: {
    // Force runes mode for application code until Svelte 6 makes it the default.
    runes: ({ filename }) => {
      if (filename.split(/[/\\]/).includes("node_modules")) {
        return undefined;
      }

      return true;
    },
  },
  kit: {
    adapter: adapter({
      fallback: "index.html",
    }),
    paths: {
      relative: false,
    },
  },
};

export default config;
