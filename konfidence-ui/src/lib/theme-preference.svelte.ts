import { browser } from "$app/environment";
import { createContext } from "svelte";
import { THEME_COOKIE, type ThemeMode, type UITheme, isThemeMode, resolveTheme } from "$lib/theme";

class ThemePreference {
  selected = $state<ThemeMode>("konfidence");
  #prefersDark = $state(false);

  constructor(initial: ThemeMode) {
    this.selected = initial;
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
      let secure = "";
      if (globalThis.location.protocol === "https:") {
        secure = "; Secure";
      }
      globalThis.document.cookie = `${THEME_COOKIE}=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
    }
  }

  #apply(): void {
    globalThis.document.documentElement.dataset.theme = this.resolved;
  }
}

const [getThemePreference, setThemePreference] = createContext<ThemePreference>();

export { getThemePreference, setThemePreference, ThemePreference };
