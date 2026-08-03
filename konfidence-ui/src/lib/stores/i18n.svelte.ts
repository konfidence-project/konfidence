import parseProperties from "@ui5/webcomponents-base/dist/PropertiesFileFormat.js";
import {
  setFetchDefaultLanguage,
  setLanguage,
} from "@ui5/webcomponents-base/dist/config/Language.js";
import { getI18nBundle, registerI18nLoader } from "@ui5/webcomponents-base/dist/i18nBundle.js";

// Import the source properties files at build time so English is available
// synchronously (avoids a flash of translation keys before the async loader
// resolves, and lets tests run without an HTTP fetch).
const propertiesModules = import.meta.glob("../i18n/messagebundle_*.properties", {
  eager: true,
  import: "default",
  query: "?raw",
}) as Record<string, string>;

interface I18nText {
  defaultText: string;
  key: string;
}

interface I18nBundleLike {
  getText: (key: I18nText | string, ...params: (number | string)[]) => string;
}

const BUNDLE_ID = "konfidence";
const STORAGE_KEY = "konfidence.ui.language";
const LOCALE_PATTERN = /messagebundle_(?<locale>[a-z-]+)\.properties$/;
const PLACEHOLDER_PATTERN = /\{(?<index>\d+)\}/g;

const SUPPORTED_LANGUAGES = [
  { id: "en", label: "LANG_EN" },
  { id: "de", label: "LANG_DE" },
] as const;

type LanguageId = (typeof SUPPORTED_LANGUAGES)[number]["id"];

const LANGUAGE_MODES = ["system", "en", "de"] as const;
type LanguageMode = (typeof LANGUAGE_MODES)[number];

const DEFAULT_LANGUAGE: LanguageId = "en";

const isLanguageId = (value: string): value is LanguageId =>
  SUPPORTED_LANGUAGES.some((entry) => entry.id === value);

const isLanguageMode = (value: string | undefined): value is LanguageMode =>
  value !== undefined && (LANGUAGE_MODES as readonly string[]).includes(value);

const parsedBundles: Record<LanguageId, Record<string, string>> = { de: {}, en: {} };
for (const [path, raw] of Object.entries(propertiesModules)) {
  const locale = LOCALE_PATTERN.exec(path)?.groups?.locale;
  if (locale && isLanguageId(locale)) {
    parsedBundles[locale] = parseProperties(raw) as Record<string, string>;
  }
}

const registeredLoaders = new Set<LanguageId>();
const ensureLoadersRegistered = (): void => {
  for (const { id } of SUPPORTED_LANGUAGES) {
    if (!registeredLoaders.has(id)) {
      registeredLoaders.add(id);
      registerI18nLoader(BUNDLE_ID, id, async () => parsedBundles[id]);
    }
  }
};

const detectBrowserLanguage = (): LanguageId => {
  const raw = globalThis.navigator?.language?.toLowerCase() ?? "";
  if (raw.startsWith("de")) {
    return "de";
  }
  return "en";
};

const resolveLanguage = (mode: LanguageMode): LanguageId => {
  if (mode === "system") {
    return detectBrowserLanguage();
  }
  return mode;
};

const loadMode = (): LanguageMode => {
  const stored = globalThis.localStorage?.getItem(STORAGE_KEY) ?? undefined;
  if (isLanguageMode(stored)) {
    return stored;
  }
  return "system";
};

const initialMode = loadMode();
const languagePreference = $state<{
  mode: LanguageMode;
  resolved: LanguageId;
  bundleVersion: number;
}>({
  bundleVersion: 0,
  mode: initialMode,
  resolved: resolveLanguage(initialMode),
});

let currentBundle: I18nBundleLike | undefined = undefined;
let initialized = false;

const applyLanguage = async (resolved: LanguageId): Promise<void> => {
  await setLanguage(resolved);
  currentBundle = (await getI18nBundle(BUNDLE_ID)) as unknown as I18nBundleLike;
  languagePreference.bundleVersion += 1;
};

const formatFallback = (
  template: string | undefined,
  key: string,
  params: (number | string)[],
): string => {
  const source = template ?? key;
  if (params.length === 0) {
    return source;
  }
  return source.replaceAll(PLACEHOLDER_PATTERN, (_match, index: string) => {
    const value = params[Number(index)];
    if (value === undefined) {
      return `{${index}}`;
    }
    return String(value);
  });
};

// oxlint-disable-next-line id-length -- `t` is the conventional i18n API name
const t = (key: string, ...params: (number | string)[]): string => {
  // Register a read on bundleVersion so Svelte reruns this call on language change.
  // eslint-disable-next-line no-unused-expressions -- reactive dependency marker
  languagePreference.bundleVersion;
  const englishFallback = parsedBundles[DEFAULT_LANGUAGE]?.[key];
  if (currentBundle) {
    // Pass a defaultText so UI5 falls back to the English source (never the raw key)
    // when the bundle content is missing for any reason.
    return currentBundle.getText({ defaultText: englishFallback ?? key, key }, ...params);
  }
  // Bundle not yet resolved; fall back to the language-specific parsed source,
  // then to the English source, then to the raw key.
  const source = parsedBundles[languagePreference.resolved]?.[key] ?? englishFallback;
  return formatFallback(source, key, params);
};

const persistMode = (mode: LanguageMode): void => {
  globalThis.localStorage?.setItem(STORAGE_KEY, mode);
};

const systemLanguageListener = (): void => {
  if (languagePreference.mode !== "system") {
    return;
  }
  const nextResolved = detectBrowserLanguage();
  if (nextResolved !== languagePreference.resolved) {
    languagePreference.resolved = nextResolved;
    void applyLanguage(nextResolved);
  }
};

const initI18n = (): void => {
  if (initialized) {
    return;
  }
  initialized = true;

  // UI5 skips its network loader when the resolved language matches its
  // built-in default (English). Force fetching so our custom loader always runs.
  setFetchDefaultLanguage(true);

  ensureLoadersRegistered();

  void applyLanguage(languagePreference.resolved);

  globalThis.addEventListener?.("languagechange", systemLanguageListener);

  $effect.root(() => {
    $effect(() => {
      const { mode } = languagePreference;
      persistMode(mode);
      const nextResolved = resolveLanguage(mode);
      if (nextResolved !== languagePreference.resolved) {
        languagePreference.resolved = nextResolved;
      }
      void applyLanguage(nextResolved);
    });
  });
};

const selectLanguage = (mode: string): void => {
  if (!isLanguageMode(mode)) {
    return;
  }
  languagePreference.mode = mode;
};

export {
  BUNDLE_ID,
  DEFAULT_LANGUAGE,
  initI18n,
  languagePreference,
  LANGUAGE_MODES,
  selectLanguage,
  SUPPORTED_LANGUAGES,
  t,
};
export type { LanguageId, LanguageMode };
