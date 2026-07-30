import type { Handle, RequestEvent } from "@sveltejs/kit";
import { resolveSession } from "$lib/server/auth";
import { LOCALE_COOKIE, type Locale, resolveLocale } from "$lib/locale";
import { THEME_COOKIE, parseThemeMode, resolveTheme } from "$lib/theme";

const setRequestLocale = (event: RequestEvent): Locale => {
  const locale = resolveLocale(
    event.cookies.get(LOCALE_COOKIE),
    event.request.headers.get("accept-language"),
  );
  event.locals.locale = locale;
  return locale;
};

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit expects this named export.
export const handle: Handle = async ({ event, resolve }) => {
  const startedAt = Date.now();
  const locale = setRequestLocale(event);

  if (event.route.id?.startsWith("/(app)") === true) {
    const session = await resolveSession(event);
    event.locals.session = session;
    event.locals.user = session?.user;
  }

  const theme = resolveTheme(parseThemeMode(event.cookies.get(THEME_COOKIE)), false);
  const response = await resolve(event, {
    transformPageChunk: ({ html }) =>
      html.replace('<html lang="en"', `<html lang="${locale}" data-theme="${theme}"`),
  });

  // oxlint-disable-next-line no-undef -- Console output is intentional server-side request diagnostics.
  console.info("[request]", {
    authenticated: Boolean(event.locals.user),
    durationMs: Date.now() - startedAt,
    method: event.request.method,
    path: event.url.pathname,
    roles: event.locals.user?.roles,
    route: event.route.id,
    status: response.status,
  });

  return response;
};
