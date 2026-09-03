import type { Theme } from "./types.js";

/** All Konfidence theme identifiers, in canonical order. */
const THEMES: readonly Theme[] = ["konfidence", "konfidence-dark"] as const;

/** Fallback theme when nothing else has been chosen. */
const DEFAULT_THEME: Theme = "konfidence";

/** `localStorage` key that persists the user's theme preference. */
const THEME_STORAGE_KEY = "konfidence.theme";

/**
 * URL query-parameter name that overrides the persisted theme on load.
 * When present with a valid value, the theme is applied AND written to
 * `localStorage` as the new preference, then the parameter is stripped
 * from the URL via `history.replaceState` so refreshes are idempotent.
 */
const THEME_QUERY_PARAM = "theme";

/** Runtime guard: is `value` one of the known Konfidence themes? */
const isTheme = (value: string | null | undefined): value is Theme =>
  value === "konfidence" || value === "konfidence-dark";

export { DEFAULT_THEME, isTheme, THEME_QUERY_PARAM, THEME_STORAGE_KEY, THEMES };
