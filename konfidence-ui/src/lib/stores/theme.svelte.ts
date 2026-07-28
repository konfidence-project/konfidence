import { createContext } from "svelte";
import { THEME_COOKIE, type UITheme, isUITheme } from "$lib/theme";

const persistTheme = (value: UITheme): void => {
  if (!globalThis.document) {
    return;
  }
  let secure = "";
  if (globalThis.location?.protocol === "https:") {
    secure = "; Secure";
  }
  globalThis.document.cookie = `${THEME_COOKIE}=${value}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
};

class ThemePreference {
  selected = $state<UITheme>("konfidence");

  constructor(getInitialTheme: () => UITheme) {
    this.selected = getInitialTheme();
  }

  select(value: string): void {
    if (!isUITheme(value)) {
      return;
    }

    this.selected = value;
    const root = globalThis.document?.documentElement;
    root?.setAttribute("data-theme", value);
    root?.classList.toggle("dark", value === "konfidence-dark");
    persistTheme(value);
  }
}

const [getThemePreference, setThemePreference] = createContext<ThemePreference>();

export { getThemePreference, setThemePreference, ThemePreference };
