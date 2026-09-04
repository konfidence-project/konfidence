/** A Konfidence theme identifier. Only `konfidence` for now; kept as a
 * discriminated string so future themes (e.g. `apeiro`) can be added
 * without changing every consumer. */
type Theme = "konfidence";

/** A colour-mode preference. `system` follows the OS via
 * `prefers-color-scheme` in CSS and via `matchMedia` at runtime. */
type Mode = "light" | "dark" | "system";

export type { Mode, Theme };
