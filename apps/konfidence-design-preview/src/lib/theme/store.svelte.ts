import { resolveInitialTheme } from "./resolve.js";
import {
  DEFAULT_MODE,
  DEFAULT_THEME,
  MODE_STORAGE_KEY,
  THEME_STORAGE_KEY,
  isMode,
  isTheme,
} from "./constants.js";
import type { Mode, Theme } from "./types.js";

/**
 * Reactive holder for the active Konfidence theme + colour mode.
 *
 * The initial values are resolved via {@link resolveInitialTheme} on
 * construction (in a browser context) so `?theme=…` / `?mode=…`
 * deep-links land on the correct pair even before the `<html
 * data-theme>` / `<html data-mode>` attributes have been re-read.
 * Subsequent calls to {@link ThemeStore.setMode} update the attribute
 * AND persist to `localStorage`.
 */
class ThemeStore {
  #theme = $state<Theme>(DEFAULT_THEME);
  #mode = $state<Mode>(DEFAULT_MODE);

  constructor() {
    if (typeof globalThis.document === "undefined") {
      return;
    }
    // The inline script in `app.html` has already resolved the pair and
    // applied them to <html data-theme>/<html data-mode>. Prefer those
    // values; fall back to resolving from location/localStorage in case
    // the attributes were stripped (e.g. by third-party tooling).
    const themeAttr = globalThis.document.documentElement.getAttribute("data-theme");
    const modeAttr = globalThis.document.documentElement.getAttribute("data-mode");
    if (isTheme(themeAttr) && isMode(modeAttr)) {
      this.#theme = themeAttr;
      this.#mode = modeAttr;
      return;
    }
    if (typeof globalThis.location !== "undefined") {
      const resolved = resolveInitialTheme({
        history: typeof globalThis.history === "undefined" ? undefined : globalThis.history,
        location: globalThis.location,
        storage:
          typeof globalThis.localStorage === "undefined" ? undefined : globalThis.localStorage,
      });
      this.#theme = resolved.theme;
      this.#mode = resolved.mode;
    }
  }

  get theme(): Theme {
    return this.#theme;
  }

  get mode(): Mode {
    return this.#mode;
  }

  setTheme(theme: Theme): void {
    this.#theme = theme;
    if (typeof globalThis.document !== "undefined") {
      globalThis.document.documentElement.setAttribute("data-theme", theme);
    }
    this.#persist(THEME_STORAGE_KEY, theme);
  }

  setMode(mode: Mode): void {
    this.#mode = mode;
    if (typeof globalThis.document !== "undefined") {
      globalThis.document.documentElement.setAttribute("data-mode", mode);
    }
    this.#persist(MODE_STORAGE_KEY, mode);
  }

  /** Cycle light -> dark -> system -> light. */
  toggleMode(): void {
    const NEXT_MODE: Record<Mode, Mode> = {
      dark: "system",
      light: "dark",
      system: "light",
    };
    this.setMode(NEXT_MODE[this.#mode]);
  }

  #persist(key: string, value: string): void {
    if (typeof globalThis.localStorage === "undefined") {
      return;
    }
    try {
      globalThis.localStorage.setItem(key, value);
    } catch {
      // LocalStorage may be disabled or full; ignore.
    }
  }
}

/** Shared instance for consumers that do not need their own store. */
const themeStore = new ThemeStore();

export { ThemeStore, themeStore };
