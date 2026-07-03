import adapter from "@sveltejs/adapter-node";

const config = {
  compilerOptions: {
    // Force runes mode for the project, except for libraries. Can be removed in svelte 6.
    runes: ({ filename }) => (filename.split(/[/\\]/).includes("node_modules") ? undefined : true),

    experimental: {
      async: true,
    },
  },
  kit: {
    adapter: adapter(),
    experimental: {
      remoteFunctions: true,
    },
  },
};

export default config;
