const DEFAULT_LOCALE = "en";
const LOCALE_COOKIE = "konfidence_locale";
const locales = [
  { id: "en", label: "English" },
  { id: "de", label: "Deutsch" },
] as const;

type Locale = (typeof locales)[number]["id"];

const isLocale = (value: string | undefined): value is Locale =>
  locales.some((locale) => locale.id === value);

const parseLocale = (value: string | undefined): Locale | undefined => {
  const language = value?.trim().replaceAll("_", "-").split("-")[0]?.toLowerCase();
  if (isLocale(language)) {
    return language;
  }
  return undefined;
};

const parseQuality = (parameter: string | undefined): number => {
  if (!parameter) {
    return 1;
  }
  const quality = Number.parseFloat(parameter.trim().slice("q=".length));
  if (Number.isNaN(quality)) {
    return 0;
  }
  return quality;
};

const resolveLocale = (cookie: string | undefined, acceptLanguage: string | null): Locale => {
  const preferred = parseLocale(cookie);
  if (preferred) {
    return preferred;
  }

  const accepted = (acceptLanguage ?? "")
    .split(",")
    .map((entry, index) => {
      const [language, ...parameters] = entry.trim().split(";");
      const qualityParameter = parameters.find((parameter) => parameter.trim().startsWith("q="));
      return { index, language, quality: parseQuality(qualityParameter) };
    })
    .filter(({ quality }) => quality > 0)
    .toSorted((left, right) => right.quality - left.quality || left.index - right.index);

  for (const { language } of accepted) {
    const locale = parseLocale(language);
    if (locale) {
      return locale;
    }
  }
  return DEFAULT_LOCALE;
};

export { DEFAULT_LOCALE, isLocale, LOCALE_COOKIE, locales, parseLocale, resolveLocale };
export type { Locale };
