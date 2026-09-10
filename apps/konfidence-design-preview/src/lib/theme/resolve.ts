import {
  DEFAULT_MODE,
  DEFAULT_THEME,
  MODE_QUERY_PARAM,
  MODE_STORAGE_KEY,
  THEME_QUERY_PARAM,
  THEME_STORAGE_KEY,
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
 *
 * Kept in sync with the inline `<script>` in `app.html` — the script
 * runs synchronously before the first paint (adapter-static, no
 * server hook), and this typed function is the fallback path used by
 * `ThemeStore` when the inline script did not run (e.g. blocked by
 * CSP or reverted by third-party tooling). Unit tests cover that
 * fallback.
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

export { resolveInitialTheme };
export type { ResolvedThemeState };
