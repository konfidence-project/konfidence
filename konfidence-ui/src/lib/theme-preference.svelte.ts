import { browser } from "$app/environment";
import { createContext } from "svelte";
import { THEME_COOKIE, type UITheme, isUITheme } from "$lib/theme";

class ThemePreference {
  selected = $state<UITheme>("konfidence");

  constructor(initial: UITheme) {
    this.selected = initial;
  }

  select(value: string): void {
    if (!isUITheme(value)) {
      return;
    }

    this.selected = value;
    if (browser) {
      let secure = "";
      if (globalThis.location.protocol === "https:") {
        secure = "; Secure";
      }
      globalThis.document.documentElement.dataset.theme = value;
      globalThis.document.cookie = `${THEME_COOKIE}=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
    }
  }
}

const [getThemePreference, setThemePreference] = createContext<ThemePreference>();

export { getThemePreference, setThemePreference, ThemePreference };
