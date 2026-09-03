/**
 * Dynamic project-scoped routes cannot be prerendered because their
 * `[projectId]` segment is only known at request time. The SPA fallback in
 * `svelte.config.js` (`fallback: "index.html"`) serves them at runtime.
 */
export const prerender = false;
