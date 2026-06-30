import { getTheme, setTheme } from "@ui5/webcomponents-base/dist/config/Theme.js";

const STORAGE_KEY = "konfidence.ui.theme";

export const themes = [
  { id: "konfidence", label: "Konfidence" },
  { id: "konfidence-dark", label: "Konfidence Dark" },
  { id: "sap_horizon", label: "SAP Horizon" },
] as const;

export type UITheme = (typeof themes)[number]["id"];

function isUITheme(theme: string | null): theme is UITheme {
  return themes.some((option) => option.id === theme);
}

function load(): UITheme {
  const storedTheme = localStorage.getItem(STORAGE_KEY);

  if (isUITheme(storedTheme)) {
    return storedTheme;
  }

  const currentTheme = getTheme();
  return isUITheme(currentTheme) ? currentTheme : "konfidence";
}

export const themePreference = $state<{ selected: UITheme }>({ selected: load() });
export const theme = themePreference;

let initialized = false;
const customThemes = new Set<UITheme>(["konfidence", "konfidence-dark"]);
const loadedCustomThemes = new Set<UITheme>();

async function applySelectedTheme(selectedTheme: UITheme) {
  if (customThemes.has(selectedTheme) && !loadedCustomThemes.has(selectedTheme)) {
    return;
  }

  await setTheme(selectedTheme);
}

export function initTheme() {
  if (initialized) {
    return;
  }

  initialized = true;

  void applySelectedTheme(themePreference.selected);

  $effect.root(() => {
    $effect(() => {
      localStorage.setItem(STORAGE_KEY, themePreference.selected);
      void applySelectedTheme(themePreference.selected);
    });
  });
}

export function markCustomThemeStylesheetLoaded(loadedTheme: UITheme) {
  if (!customThemes.has(loadedTheme)) {
    return;
  }

  loadedCustomThemes.add(loadedTheme);

  if (themePreference.selected === loadedTheme) {
    void applySelectedTheme(loadedTheme);
  }
}

export function selectTheme(selectedTheme: string) {
  if (isUITheme(selectedTheme)) {
    themePreference.selected = selectedTheme;
  }
}
