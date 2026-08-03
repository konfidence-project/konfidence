const SETTINGS_TAB_IDS = ["profile", "appearance"] as const;

type SettingsTab = (typeof SETTINGS_TAB_IDS)[number];

const DEFAULT_SETTINGS_TAB: SettingsTab = "profile";

const SETTINGS_URL_PARAM = "settings";

interface SettingsTabDefinition {
  icon: string;
  id: SettingsTab;
  label: string;
}

const SETTINGS_TABS: readonly SettingsTabDefinition[] = [
  { icon: "user", id: "profile", label: "Profile" },
  { icon: "palette", id: "appearance", label: "Appearance" },
];

const isSettingsTab = (value: string): value is SettingsTab =>
  (SETTINGS_TAB_IDS as readonly string[]).includes(value);

const parseSettingsTab = (value: string | undefined): SettingsTab | undefined => {
  if (!value) {
    return undefined;
  }
  if (isSettingsTab(value)) {
    return value;
  }
  return undefined;
};

export {
  DEFAULT_SETTINGS_TAB,
  parseSettingsTab,
  SETTINGS_TABS,
  SETTINGS_URL_PARAM,
  type SettingsTab,
};
