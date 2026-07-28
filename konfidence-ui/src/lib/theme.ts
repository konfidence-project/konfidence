const THEME_COOKIE = "konfidence_theme";
const DEFAULT_THEME = "konfidence";

const themes = [
  { description: "Bright Konfidence blue", id: "konfidence", label: "Konfidence" },
  { description: "Low-light workspace", id: "konfidence-dark", label: "Konfidence Dark" },
  { description: "Classic Horizon palette", id: "sap_horizon", label: "SAP Horizon" },
] as const;

type UITheme = (typeof themes)[number]["id"];

const isUITheme = (value: string | undefined): value is UITheme =>
  themes.some((theme) => theme.id === value);

const parseTheme = (value: string | undefined): UITheme => {
  if (isUITheme(value)) {
    return value;
  }
  return DEFAULT_THEME;
};

export { DEFAULT_THEME, isUITheme, parseTheme, THEME_COOKIE, themes };
export type { UITheme };
