import adapter from "@sveltejs/adapter-static";

const config = {
  compilerOptions: {
    experimental: {
      async: true,
    },
    // Force runes mode for the project, except for libraries. Can be removed in svelte 6.
    runes: ({ filename }) => {
      if (filename.split(/[/\\]/).includes("node_modules")) {
        return undefined;
      }

      return true;
    },
  },
  kit: {
    adapter: adapter({ fallback: "index.html" }),
  },
};

export default config;
