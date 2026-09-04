import type { Handle } from "@sveltejs/kit";

import { buildInlineBootstrapScript } from "$lib/theme";

/**
 * Placeholder written into `app.html`. `transformPageChunk` replaces it
 * with the inline theme bootstrap script during prerender. Kept as an
 * HTML comment so the source file remains valid HTML that browsers
 * would tolerate even if the hook were ever bypassed.
 */
const THEME_BOOTSTRAP_MARKER = "<!--@konfidence/theme-bootstrap-->";

/**
 * Built once per build. `buildInlineBootstrapScript()` produces the
 * body of a synchronous `<script>` that applies `<html data-theme>`
 * before any stylesheet paints, preventing a theme flash on reload.
 * Injected here (rather than pasted into `app.html`) so the design
 * system stays the single source of truth for theme constants and
 * resolution logic.
 */
const themeBootstrapScript = `<script>${buildInlineBootstrapScript()}</script>`;

const handle: Handle = ({ event, resolve }) =>
  resolve(event, {
    transformPageChunk: ({ html }) => html.replace(THEME_BOOTSTRAP_MARKER, themeBootstrapScript),
  });

export { handle };
