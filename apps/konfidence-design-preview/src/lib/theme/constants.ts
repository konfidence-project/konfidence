import type { Mode, Theme } from "./types.js";

/** All Konfidence theme identifiers, in canonical order. */
const THEMES: readonly Theme[] = ["konfidence"] as const;

/** All colour modes users can pick, in canonical order. */
const MODES: readonly Mode[] = ["light", "dark", "system"] as const;

/** Fallback theme when nothing else has been chosen. */
const DEFAULT_THEME: Theme = "konfidence";

/** Fallback mode when nothing else has been chosen. */
const DEFAULT_MODE: Mode = "light";

/** `localStorage` key that persists the user's theme preference. */
const THEME_STORAGE_KEY = "konfidence.theme";

/** `localStorage` key that persists the user's colour-mode preference. */
const MODE_STORAGE_KEY = "konfidence.mode";

/**
 * URL query-parameter name that overrides the persisted theme on load.
 * When present with a valid value, the theme is applied AND written to
 * `localStorage` as the new preference, then the parameter is stripped
 * from the URL via `history.replaceState` so refreshes are idempotent.
 */
const THEME_QUERY_PARAM = "theme";

/** URL query-parameter name that overrides the persisted mode on load. */
const MODE_QUERY_PARAM = "mode";

/** Runtime guard: is `value` one of the known Konfidence themes? */
const isTheme = (value: string | null | undefined): value is Theme => value === "konfidence";

/** Runtime guard: is `value` one of the known colour modes? */
const isMode = (value: string | null | undefined): value is Mode =>
  value === "light" || value === "dark" || value === "system";

export {
  DEFAULT_MODE,
  DEFAULT_THEME,
  isMode,
  isTheme,
  MODE_QUERY_PARAM,
  MODE_STORAGE_KEY,
  MODES,
  THEME_QUERY_PARAM,
  THEME_STORAGE_KEY,
  THEMES,
};
