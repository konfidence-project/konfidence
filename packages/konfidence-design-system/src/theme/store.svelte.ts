import { resolveInitialTheme } from "./bootstrap.js";
import { DEFAULT_THEME, THEME_STORAGE_KEY, isTheme } from "./constants.js";
import type { Theme } from "./types.js";

/**
 * Reactive holder for the active Konfidence theme.
 *
 * The initial value is resolved via {@link resolveInitialTheme} on
 * construction (in a browser context) so `?theme=…` deep-links land on
 * the correct theme even before the `<html data-theme>` attribute has
 * been re-read. Subsequent calls to {@link ThemeStore.set} update the
 * attribute AND persist to `localStorage`.
 */
class ThemeStore {
  #current = $state<Theme>(DEFAULT_THEME);

  constructor() {
    if (typeof globalThis.document === "undefined") {
      return;
    }
    // The inline script in `app.html` has already resolved the theme and
    // applied it to <html data-theme>. Prefer that value; fall back to
    // resolving from location/localStorage in case the attribute was
    // stripped (e.g. by third-party tooling).
    const attr = globalThis.document.documentElement.getAttribute("data-theme");
    if (isTheme(attr)) {
      this.#current = attr;
      return;
    }
    if (typeof globalThis.location !== "undefined") {
      this.#current = resolveInitialTheme({
        history: typeof globalThis.history === "undefined" ? undefined : globalThis.history,
        location: globalThis.location,
        storage:
          typeof globalThis.localStorage === "undefined" ? undefined : globalThis.localStorage,
      });
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
        // LocalStorage may be disabled or full; ignore.
      }
    }
  }

  toggle(): void {
    this.set(this.#current === "konfidence" ? "konfidence-dark" : "konfidence");
  }
}

/** Shared instance for consumers that do not need their own store. */
const themeStore = new ThemeStore();

export { ThemeStore, themeStore };
