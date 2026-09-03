/**
 * Embedded mode. When present the shell chrome (topbar, sidebar) is hidden and
 * the page uses the full viewport. Auth, routing, and page logic keep running.
 *
 * The flag is carried in the URL so it survives client-side navigation and
 * copy-paste. `(shell)/+layout.svelte` reattaches it to outgoing internal
 * navigations via `beforeNavigate`.
 *
 * Documented in `apps/konfidence-ui/README.md#embedded-mode`.
 */
const EMBEDDED_QUERY = "embedded";
const EMBEDDED_ON = "1";

const isEmbedded = (url: URL): boolean => url.searchParams.get(EMBEDDED_QUERY) === EMBEDDED_ON;

export { EMBEDDED_QUERY, EMBEDDED_ON, isEmbedded };
