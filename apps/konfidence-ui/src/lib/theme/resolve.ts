import {
  DEFAULT_MODE,
  DEFAULT_THEME,
  MODE_QUERY_PARAM,
  MODE_STORAGE_KEY,
  MODES,
  THEME_QUERY_PARAM,
  THEME_STORAGE_KEY,
  THEMES,
  isMode,
  isTheme,
} from "./constants.js";
import type { Mode, Theme } from "./types.js";

/** The resolved `data-theme` / `data-mode` pair for the initial render. */
interface ResolvedThemeState {
  mode: Mode;
  theme: Theme;
}

/**
 * Resolve the initial theme + mode in priority order and apply any
 * required side effects (persistence, URL cleanup). Precedence, per
 * axis:
 *
 *   1. `?theme=…` / `?mode=…` in the URL — persisted, then stripped.
 *   2. `localStorage["konfidence.theme"]` / `["konfidence.mode"]` — used verbatim.
 *   3. `DEFAULT_THEME` / `DEFAULT_MODE`   — light `konfidence`.
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
}): ResolvedThemeState => {
  const { history, location, storage } = options;

  const params = new URLSearchParams(location.search);
  const queryTheme = params.get(THEME_QUERY_PARAM);
  const queryMode = params.get(MODE_QUERY_PARAM);

  const themeFromQuery = isTheme(queryTheme) ? queryTheme : undefined;
  const modeFromQuery = isMode(queryMode) ? queryMode : undefined;

  if (themeFromQuery !== undefined && storage) {
    try {
      storage.setItem(THEME_STORAGE_KEY, themeFromQuery);
    } catch {
      // LocalStorage may be disabled or full; ignore.
    }
  }
  if (modeFromQuery !== undefined && storage) {
    try {
      storage.setItem(MODE_STORAGE_KEY, modeFromQuery);
    } catch {
      // LocalStorage may be disabled or full; ignore.
    }
  }

  if ((themeFromQuery !== undefined || modeFromQuery !== undefined) && history) {
    params.delete(THEME_QUERY_PARAM);
    params.delete(MODE_QUERY_PARAM);
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

  let theme: Theme = themeFromQuery ?? DEFAULT_THEME;
  let mode: Mode = modeFromQuery ?? DEFAULT_MODE;

  if (themeFromQuery === undefined && storage) {
    try {
      const stored = storage.getItem(THEME_STORAGE_KEY);
      if (isTheme(stored)) {
        theme = stored;
      }
    } catch {
      // LocalStorage may be disabled; fall through.
    }
  }
  if (modeFromQuery === undefined && storage) {
    try {
      const stored = storage.getItem(MODE_STORAGE_KEY);
      if (isMode(stored)) {
        mode = stored;
      }
    } catch {
      // LocalStorage may be disabled; fall through.
    }
  }

  return { mode, theme };
};

/**
 * Body of the inline `<script>` injected into every prerendered HTML
 * page. Must run synchronously in `<head>` before any stylesheet paints
 * so a reload never flashes the wrong theme.
 *
 * Constants are pulled from the shared `./constants.js` module so this
 * script and {@link resolveInitialTheme} agree on theme names, the
 * storage keys, and the query-parameter names. The typed function is
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
  const modes = JSON.stringify(MODES);
  const themeQuery = JSON.stringify(THEME_QUERY_PARAM);
  const modeQuery = JSON.stringify(MODE_QUERY_PARAM);
  const themeStorage = JSON.stringify(THEME_STORAGE_KEY);
  const modeStorage = JSON.stringify(MODE_STORAGE_KEY);
  return `(() => {
  try {
    const themes = new Set(${themes});
    const modes = new Set(${modes});
    const params = new URLSearchParams(location.search);
    const queryTheme = params.get(${themeQuery});
    const queryMode = params.get(${modeQuery});
    let theme, mode;
    if (themes.has(queryTheme)) {
      theme = queryTheme;
      try { localStorage.setItem(${themeStorage}, theme); } catch {}
    } else {
      try {
        const stored = localStorage.getItem(${themeStorage});
        if (themes.has(stored)) theme = stored;
      } catch {}
    }
    if (modes.has(queryMode)) {
      mode = queryMode;
      try { localStorage.setItem(${modeStorage}, mode); } catch {}
    } else {
      try {
        const stored = localStorage.getItem(${modeStorage});
        if (modes.has(stored)) mode = stored;
      } catch {}
    }
    if (queryTheme != null || queryMode != null) {
      try {
        params.delete(${themeQuery});
        params.delete(${modeQuery});
        const search = params.toString();
        history.replaceState(
          null,
          "",
          location.pathname + (search ? "?" + search : "") + location.hash,
        );
      } catch {}
    }
    if (theme) document.documentElement.setAttribute("data-theme", theme);
    if (mode) document.documentElement.setAttribute("data-mode", mode);
  } catch {}
})();`;
};

export { buildInlineBootstrapScript, resolveInitialTheme };
export type { ResolvedThemeState };
