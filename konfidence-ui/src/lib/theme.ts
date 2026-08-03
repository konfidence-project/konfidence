const THEME_COOKIE = "konfidence_theme";

const themes = [
  { description: "Bright and focused", id: "konfidence", label: "Konfidence" },
  { description: "Low-light workspace", id: "konfidence-dark", label: "Konfidence Dark" },
  { description: "Familiar enterprise blue", id: "sap_horizon", label: "Horizon" },
] as const;

type UITheme = (typeof themes)[number]["id"];

const isUITheme = (value: string | undefined): value is UITheme =>
  themes.some((theme) => theme.id === value);

const parseTheme = (value: string | undefined): UITheme => {
  if (isUITheme(value)) {
    return value;
  }
  return "konfidence";
};

export { isUITheme, parseTheme, THEME_COOKIE, themes };
export type { UITheme };
