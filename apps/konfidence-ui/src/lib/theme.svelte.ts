/**
 * Konfidence theme store.
 *
 * Two themes are supported today, both derived from the design system:
 * - `konfidence`      — light
 * - `konfidence-dark` — dark
 *
 * The value is persisted to `localStorage` under `THEME_STORAGE_KEY` and is
 * applied to `document.documentElement` via the `data-theme` attribute. The
 * initial value is also applied by a synchronous script in `app.html` before
 * hydration so reloads never flash the wrong theme.
 *
 * This module intentionally does not ship a UI toggle. A settings dialog can
 * import {@link themeStore} and call {@link ThemeStore.set} to switch themes.
 */

type Theme = "konfidence" | "konfidence-dark";

const THEMES: readonly Theme[] = ["konfidence", "konfidence-dark"];
const DEFAULT_THEME: Theme = "konfidence";
const THEME_STORAGE_KEY = "konfidence.theme";

const isTheme = (value: string | null | undefined): value is Theme =>
  value === "konfidence" || value === "konfidence-dark";

/** Reactive holder for the active Konfidence theme. */
class ThemeStore {
  #current = $state<Theme>(DEFAULT_THEME);

  constructor() {
    if (typeof globalThis.document !== "undefined") {
      const attr = globalThis.document.documentElement.getAttribute("data-theme");
      if (isTheme(attr)) {
        this.#current = attr;
      }
    }
  }

  get current(): Theme {
    return this.#current;
  }

  set(theme: Theme): void {
    this.#current = theme;
    if (typeof globalThis.document !== "undefined") {
      globalThis.document.documentElement.setAttribute("data-theme", theme);
    }
    if (typeof globalThis.localStorage !== "undefined") {
      try {
        globalThis.localStorage.setItem(THEME_STORAGE_KEY, theme);
      } catch {
        /* localStorage may be disabled or full; ignore */
      }
    }
  }

  toggle(): void {
    if (this.#current === "konfidence") {
      this.set("konfidence-dark");
    } else {
      this.set("konfidence");
    }
  }
}

/** Shared instance for consumers that do not need their own store. */
const themeStore = new ThemeStore();

export type { Theme };
export { DEFAULT_THEME, THEME_STORAGE_KEY, THEMES, ThemeStore, themeStore };
