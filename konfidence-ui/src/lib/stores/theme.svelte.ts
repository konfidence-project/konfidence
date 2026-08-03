import { getTheme, setTheme } from "@ui5/webcomponents-base/dist/config/Theme.js";

const STORAGE_KEY = "konfidence.ui.theme";
const MODE_STORAGE_KEY = "konfidence.ui.themeMode";
const themes = [
  { id: "konfidence", label: "Konfidence" },
  { id: "konfidence-dark", label: "Konfidence Dark" },
  { id: "sap_horizon", label: "SAP Horizon" },
] as const;

type UITheme = (typeof themes)[number]["id"];

const THEME_MODES = ["system", "light", "dark", "custom"] as const;
type ThemeMode = (typeof THEME_MODES)[number];

const isUITheme = (theme: string | undefined): theme is UITheme =>
  themes.some((option) => option.id === theme);

const isThemeMode = (value: string | undefined): value is ThemeMode =>
  value !== undefined && (THEME_MODES as readonly string[]).includes(value);

const KONFIDENCE_LIGHT: UITheme = "konfidence";
const KONFIDENCE_DARK: UITheme = "konfidence-dark";

const prefersDarkMedia = (): MediaQueryList | undefined =>
  globalThis.matchMedia?.("(prefers-color-scheme: dark)");

const resolvedSystemTheme = (): UITheme => {
  if (prefersDarkMedia()?.matches) {
    return KONFIDENCE_DARK;
  }
  return KONFIDENCE_LIGHT;
};

const themeToMode = (theme: UITheme): ThemeMode => {
  if (theme === KONFIDENCE_LIGHT) {
    return "light";
  }
  if (theme === KONFIDENCE_DARK) {
    return "dark";
  }
  return "custom";
};

const themeForMode = (mode: ThemeMode): UITheme | undefined => {
  if (mode === "light") {
    return KONFIDENCE_LIGHT;
  }
  if (mode === "dark") {
    return KONFIDENCE_DARK;
  }
  if (mode === "system") {
    return resolvedSystemTheme();
  }
  return undefined;
};

const loadTheme = (): UITheme => {
  const storedTheme = globalThis.localStorage?.getItem(STORAGE_KEY) ?? undefined;

  if (isUITheme(storedTheme)) {
    return storedTheme;
  }

  const currentTheme = getTheme();
  if (isUITheme(currentTheme)) {
    return currentTheme;
  }

  return "konfidence";
};

const loadMode = (initialTheme: UITheme): ThemeMode => {
  const stored = globalThis.localStorage?.getItem(MODE_STORAGE_KEY) ?? undefined;
  if (isThemeMode(stored)) {
    return stored;
  }
  return themeToMode(initialTheme);
};

const initialTheme = loadTheme();

const themePreference = $state<{ selected: UITheme }>({ selected: initialTheme });
const theme = themePreference;
const themeModePreference = $state<{ selected: ThemeMode }>({
  selected: loadMode(initialTheme),
});

let initialized = false;

const persistTheme = (selectedTheme: UITheme): void => {
  globalThis.localStorage?.setItem(STORAGE_KEY, selectedTheme);
};

const persistMode = (selectedMode: ThemeMode): void => {
  globalThis.localStorage?.setItem(MODE_STORAGE_KEY, selectedMode);
};

const applyTheme = (selectedTheme: UITheme): void => {
  themePreference.selected = selectedTheme;
  persistTheme(selectedTheme);
  void setTheme(selectedTheme);
};

const systemPreferenceListener = (event: MediaQueryListEvent): void => {
  if (themeModePreference.selected !== "system") {
    return;
  }
  let nextTheme: UITheme = KONFIDENCE_LIGHT;
  if (event.matches) {
    nextTheme = KONFIDENCE_DARK;
  }
  if (themePreference.selected !== nextTheme) {
    applyTheme(nextTheme);
  }
};

const initTheme = (): void => {
  if (initialized) {
    return;
  }

  initialized = true;

  const resolvedInitialTheme =
    themeForMode(themeModePreference.selected) ?? themePreference.selected;
  applyTheme(resolvedInitialTheme);
  persistMode(themeModePreference.selected);

  prefersDarkMedia()?.addEventListener?.("change", systemPreferenceListener);
};

// Kept as a no-op for backwards compatibility with the root layout's onload
// callback. The theme stylesheet media flip is handled reactively; explicit
// gating on stylesheet load caused switch-to-dark to silently no-op in browsers
// that skip firing onload for media-mismatched stylesheets.
const markCustomThemeStylesheetLoaded = (_loadedTheme: UITheme): void => {
  void _loadedTheme;
};

const selectTheme = (selectedTheme: string): void => {
  if (!isUITheme(selectedTheme)) {
    return;
  }
  applyTheme(selectedTheme);
  const nextMode = themeToMode(selectedTheme);
  if (themeModePreference.selected !== nextMode) {
    themeModePreference.selected = nextMode;
    persistMode(nextMode);
  }
};

const selectThemeMode = (mode: string): void => {
  if (!isThemeMode(mode)) {
    return;
  }
  themeModePreference.selected = mode;
  persistMode(mode);
  const nextTheme = themeForMode(mode);
  if (nextTheme && themePreference.selected !== nextTheme) {
    applyTheme(nextTheme);
  }
};

const resolveCustomMode = (): "light" | "dark" => {
  if (themePreference.selected.includes("dark")) {
    return "dark";
  }
  return "light";
};

const resolveSystemMode = (): "light" | "dark" => {
  if (prefersDarkMedia()?.matches) {
    return "dark";
  }
  return "light";
};

const resolvedThemeMode = (): "light" | "dark" => {
  const mode = themeModePreference.selected;
  if (mode === "light") {
    return "light";
  }
  if (mode === "dark") {
    return "dark";
  }
  if (mode === "system") {
    return resolveSystemMode();
  }
  return resolveCustomMode();
};

export {
  initTheme,
  markCustomThemeStylesheetLoaded,
  resolvedThemeMode,
  selectTheme,
  selectThemeMode,
  theme,
  themeModePreference,
  themePreference,
  THEME_MODES,
  themes,
};
export type { ThemeMode, UITheme };
