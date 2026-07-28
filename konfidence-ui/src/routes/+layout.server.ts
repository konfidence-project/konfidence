import { THEME_COOKIE, parseTheme } from "$lib/theme";
import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = ({ cookies, locals }) => ({
  locale: locals.locale,
  theme: parseTheme(cookies.get(THEME_COOKIE)),
});
