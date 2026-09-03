import {
  DEFAULT_THEME,
  THEMES,
  THEME_QUERY_PARAM,
  THEME_STORAGE_KEY,
  isTheme,
} from "./constants.js";
import type { Theme } from "./types.js";

/**
 * Resolve the initial theme in priority order and apply any required
 * side effects (persistence, URL cleanup). Precedence:
 *
 *   1. `?theme=konfidence|konfidence-dark` — persisted, then stripped.
 *   2. `localStorage["konfidence.theme"]`  — used verbatim.
 *   3. `DEFAULT_THEME`                     — light.
 *
 * The helper is pure with respect to its inputs and touches only the
 * `location`, `history`, and `localStorage` you pass in, which keeps
 * it usable from tests without a full DOM.
 */
// eslint-disable-next-line max-statements
const resolveInitialTheme = (options: {
  location: Pick<Location, "search" | "pathname" | "hash">;
  history?: Pick<History, "replaceState">;
  storage?: Pick<Storage, "getItem" | "setItem">;
}): Theme => {
  const { history, location, storage } = options;

  const params = new URLSearchParams(location.search);
  const queryTheme = params.get(THEME_QUERY_PARAM);
  if (isTheme(queryTheme)) {
    if (storage) {
      try {
        storage.setItem(THEME_STORAGE_KEY, queryTheme);
      } catch {
        // LocalStorage may be disabled or full; ignore.
      }
    }
    if (history) {
      params.delete(THEME_QUERY_PARAM);
      const query = params.toString();
      const cleaned = location.pathname + (query ? `?${query}` : "") + location.hash;
      try {
        // History.replaceState explicitly accepts null for the state slot;
        // undefined would serialise as the string "undefined".
        // eslint-disable-next-line unicorn/no-null
        history.replaceState(null, "", cleaned);
      } catch {
        // ReplaceState can throw in cross-origin sandboxes; ignore.
      }
    }
    return queryTheme;
  }

  if (storage) {
    try {
      const stored = storage.getItem(THEME_STORAGE_KEY);
      if (isTheme(stored)) {
        return stored;
      }
    } catch {
      // LocalStorage may be disabled; fall through.
    }
  }

  return DEFAULT_THEME;
};

/**
 * Body of the inline `<script>` injected into every prerendered HTML
 * page. Must run synchronously in `<head>` before any stylesheet paints
 * so a reload never flashes the wrong theme.
 *
 * Constants are pulled from the shared `./constants.js` module so this
 * script and {@link resolveInitialTheme} agree on theme names, the
 * storage key, and the query-parameter name. The typed function is
 * still exercised by the `ThemeStore` as a fallback when the inline
 * script did not run (e.g. blocked by CSP) or its side effects were
 * reverted by third-party tooling; unit tests cover that path.
 *
 * Injected at build time by `apps/konfidence-ui/src/hooks.server.ts`
 * via `transformPageChunk`, so `app.html` stays free of copy-pasted
 * script bodies.
 */
const buildInlineBootstrapScript = (): string => {
  const themes = JSON.stringify(THEMES);
  const queryParam = JSON.stringify(THEME_QUERY_PARAM);
  const storageKey = JSON.stringify(THEME_STORAGE_KEY);
  return `(() => {
  try {
    const themes = new Set(${themes});
    const query = new URLSearchParams(location.search).get(${queryParam});
    let theme;
    if (themes.has(query)) {
      theme = query;
      try { localStorage.setItem(${storageKey}, theme); } catch {}
      try {
        const params = new URLSearchParams(location.search);
        params.delete(${queryParam});
        const search = params.toString();
        history.replaceState(
          null,
          "",
          location.pathname + (search ? "?" + search : "") + location.hash,
        );
      } catch {}
    } else {
      try {
        const stored = localStorage.getItem(${storageKey});
        if (themes.has(stored)) theme = stored;
      } catch {}
    }
    if (theme) document.documentElement.setAttribute("data-theme", theme);
  } catch {}
})();`;
};

export { buildInlineBootstrapScript, resolveInitialTheme };
