declare module "virtual:playground-runtime" {
  const runtime: string;
  export default runtime;
}

// Generated Svelte code imports this untyped internal runtime. It is bundled
// without being called by handwritten application code.
declare module "svelte/internal/client";
