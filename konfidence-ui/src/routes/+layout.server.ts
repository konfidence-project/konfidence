import { THEME_COOKIE, parseThemeMode } from "$lib/theme";
import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = ({ cookies, locals }) => ({
  locale: locals.locale,
  theme: parseThemeMode(cookies.get(THEME_COOKIE)),
});
