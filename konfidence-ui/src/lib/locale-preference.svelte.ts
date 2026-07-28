import { browser } from "$app/environment";
import { createContext } from "svelte";
import { get } from "svelte/store";
import { locale as i18nLocale, t } from "svelte-i18n";
import { LOCALE_COOKIE, type Locale, isLocale } from "$lib/locale";

class LocalePreference {
  selected = $state<Locale>("en");

  constructor(initial: Locale) {
    this.selected = initial;
    i18nLocale.set(initial);
  }

  translate(id: string, values?: Record<string, number | string>): string {
    return get(t)(id, { locale: this.selected, values });
  }

  select(value: string): void {
    if (!isLocale(value)) {
      return;
    }

    this.selected = value;
    i18nLocale.set(value);
    if (browser) {
      let secure = "";
      if (globalThis.location.protocol === "https:") {
        secure = "; Secure";
      }
      globalThis.document.documentElement.lang = value;
      globalThis.document.cookie = `${LOCALE_COOKIE}=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
    }
  }
}

const [getLocalePreference, setLocalePreference] = createContext<LocalePreference>();

export { getLocalePreference, LocalePreference, setLocalePreference };
