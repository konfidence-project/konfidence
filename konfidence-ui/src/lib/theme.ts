const THEME_COOKIE = "konfidence_theme";
const DEFAULT_THEME = "konfidence";

const themes = [
  { description: "Bright Konfidence blue", id: "konfidence", label: "Konfidence" },
  { description: "Low-light workspace", id: "konfidence-dark", label: "Konfidence Dark" },
  { description: "Classic Horizon palette", id: "sap_horizon", label: "SAP Horizon" },
] as const;

type UITheme = (typeof themes)[number]["id"];

const SYSTEM_THEME = "system";
type ThemeMode = UITheme | typeof SYSTEM_THEME;

// Concrete themes applied when following the operating system preference.
const SYSTEM_LIGHT_THEME: UITheme = "konfidence";
const SYSTEM_DARK_THEME: UITheme = "konfidence-dark";

const themeModes = [
  { description: "Bright Konfidence blue", id: "konfidence", label: "Konfidence" },
  { description: "Low-light workspace", id: "konfidence-dark", label: "Konfidence Dark" },
  { description: "Classic Horizon palette", id: "sap_horizon", label: "SAP Horizon" },
  { description: "Match your OS preference", id: SYSTEM_THEME, label: "System" },
] as const;

const isUITheme = (value: string | undefined): value is UITheme =>
  themes.some((theme) => theme.id === value);

const isThemeMode = (value: string | undefined): value is ThemeMode =>
  value === SYSTEM_THEME || isUITheme(value);

const parseTheme = (value: string | undefined): ThemeMode => {
  if (isThemeMode(value)) {
    return value;
  }
  return DEFAULT_THEME;
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

export {
  DEFAULT_THEME,
  isThemeMode,
  isUITheme,
  parseTheme,
  resolveTheme,
  SYSTEM_THEME,
  THEME_COOKIE,
  themeModes,
  themes,
};
export type { ThemeMode, UITheme };
