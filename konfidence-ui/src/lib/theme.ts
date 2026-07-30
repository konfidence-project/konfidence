const THEME_COOKIE = "konfidence_theme";

const themes = [
  { description: "Bright and focused", id: "konfidence", label: "Konfidence" },
  { description: "Low-light workspace", id: "konfidence-dark", label: "Konfidence Dark" },
  { description: "Familiar enterprise blue", id: "sap_horizon", label: "Horizon" },
] as const;

type UITheme = (typeof themes)[number]["id"];

const SYSTEM_THEME = "system";
type ThemeMode = UITheme | typeof SYSTEM_THEME;

// Concrete themes applied when following the operating system preference.
const SYSTEM_LIGHT_THEME: UITheme = "konfidence";
const SYSTEM_DARK_THEME: UITheme = "konfidence-dark";

const isUITheme = (value: string | undefined): value is UITheme =>
  themes.some((theme) => theme.id === value);

const isThemeMode = (value: string | undefined): value is ThemeMode =>
  value === SYSTEM_THEME || isUITheme(value);

const parseThemeMode = (value: string | undefined): ThemeMode => {
  if (isThemeMode(value)) {
    return value;
  }
  return "konfidence";
};

const resolveTheme = (mode: ThemeMode, prefersDark: boolean): UITheme => {
  if (mode !== SYSTEM_THEME) {
    return mode;
  }
  if (prefersDark) {
    return SYSTEM_DARK_THEME;
  }
  return SYSTEM_LIGHT_THEME;
};

export { isThemeMode, isUITheme, parseThemeMode, resolveTheme, SYSTEM_THEME, THEME_COOKIE, themes };
export type { ThemeMode, UITheme };
