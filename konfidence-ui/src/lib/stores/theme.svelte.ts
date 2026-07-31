import { browser } from "$app/environment";
import { createContext } from "svelte";
import {
  DEFAULT_THEME,
  THEME_COOKIE,
  type ThemeMode,
  type UITheme,
  isThemeMode,
  resolveTheme,
} from "$lib/theme";

const persistTheme = (value: ThemeMode): void => {
  if (!browser) {
    return;
  }
  let secure = "";
  if (globalThis.location?.protocol === "https:") {
    secure = "; Secure";
  }
  globalThis.document.cookie = `${THEME_COOKIE}=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
};

class ThemePreference {
  selected = $state<ThemeMode>(DEFAULT_THEME);
  #prefersDark = $state(false);

  constructor(getInitialTheme: () => ThemeMode) {
    this.selected = getInitialTheme();
    if (browser) {
      const query = globalThis.matchMedia("(prefers-color-scheme: dark)");
      this.#prefersDark = query.matches;
      query.addEventListener("change", (event) => {
        this.#prefersDark = event.matches;
        this.#apply();
      });
      this.#apply();
    }
  }

  get resolved(): UITheme {
    return resolveTheme(this.selected, this.#prefersDark);
  }

  select(value: string): void {
    if (!isThemeMode(value)) {
      return;
    }

    this.selected = value;
    if (browser) {
      this.#apply();
    }
    persistTheme(value);
  }

  #apply(): void {
    const root = globalThis.document?.documentElement;
    if (!root) {
      return;
    }
    const { resolved } = this;
    root.setAttribute("data-theme", resolved);
    root.classList.toggle("dark", resolved === "konfidence-dark");
  }
}

const [getThemePreference, setThemePreference] = createContext<ThemePreference>();

export { getThemePreference, setThemePreference, ThemePreference };
