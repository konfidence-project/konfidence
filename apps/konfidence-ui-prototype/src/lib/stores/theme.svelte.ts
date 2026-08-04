import { getTheme, setTheme } from "@ui5/webcomponents-base/dist/config/Theme.js";

const STORAGE_KEY = "konfidence.ui.theme";
const themes = [
  { id: "konfidence", label: "Konfidence" },
  { id: "konfidence-dark", label: "Konfidence Dark" },
  { id: "sap_horizon", label: "SAP Horizon" },
] as const;

type UITheme = (typeof themes)[number]["id"];

const isUITheme = (theme: string | undefined): theme is UITheme =>
  themes.some((option) => option.id === theme);

const load = (): UITheme => {
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

const themePreference = $state<{ selected: UITheme }>({ selected: load() });
const theme = themePreference;

let initialized = false;
const customThemes = new Set<UITheme>(["konfidence", "konfidence-dark"]);
const loadedCustomThemes = new Set<UITheme>();

const applySelectedTheme = async (selectedTheme: UITheme) => {
  if (customThemes.has(selectedTheme) && !loadedCustomThemes.has(selectedTheme)) {
    return;
  }

  await setTheme(selectedTheme);
};

const initTheme = () => {
  if (initialized) {
    return;
  }

  initialized = true;

  void applySelectedTheme(themePreference.selected);

  $effect.root(() => {
    $effect(() => {
      globalThis.localStorage?.setItem(STORAGE_KEY, themePreference.selected);
      void applySelectedTheme(themePreference.selected);
    });
  });
};

const markCustomThemeStylesheetLoaded = (loadedTheme: UITheme) => {
  if (!customThemes.has(loadedTheme)) {
    return;
  }

  loadedCustomThemes.add(loadedTheme);

  if (themePreference.selected === loadedTheme) {
    void applySelectedTheme(loadedTheme);
  }
};

const selectTheme = (selectedTheme: string) => {
  if (isUITheme(selectedTheme)) {
    themePreference.selected = selectedTheme;
  }
};

export { initTheme, markCustomThemeStylesheetLoaded, selectTheme, theme, themePreference, themes };
export type { UITheme };
