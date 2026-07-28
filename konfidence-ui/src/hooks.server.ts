import { type Handle, redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import { resolveSession } from "$lib/server/auth";
import { SYSTEM_THEME, THEME_COOKIE, parseTheme } from "$lib/theme";
import { getTextDirection } from "$lib/paraglide/runtime.js";
import { paraglideMiddleware } from "$lib/paraglide/server.js";

const localizedHandle: Handle = ({ event, resolve }) =>
  paraglideMiddleware(event.request, async ({ request, locale }) => {
    const startedAt = Date.now();
    event.request = request;

    if (event.route.id?.startsWith("/(app)") === true) {
      const session = await resolveSession(event);
      event.locals.session = session;
      event.locals.user = session?.user;
      if (!session) {
        const returnTo = event.url.pathname + event.url.search;
        redirect(HTTP_SEE_OTHER, `/api/login?returnTo=${encodeURIComponent(returnTo)}`);
      }
    }

    const theme = parseTheme(event.cookies.get(THEME_COOKIE));
    // "system" is resolved client-side by an inline script in app.html to avoid a flash.
    let renderedTheme: string = theme;
    if (theme === SYSTEM_THEME) {
      renderedTheme = "konfidence";
    }
    let themeClass = "";
    if (renderedTheme === "konfidence-dark") {
      themeClass = "dark";
    }
    const response = await resolve(event, {
      transformPageChunk: ({ html }) =>
        html
          .replace("%konfidence.lang%", locale)
          .replace("%konfidence.dir%", getTextDirection(locale))
          .replace("%konfidence.theme%", renderedTheme)
          .replace("%konfidence.theme-class%", themeClass),
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
  });

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit expects this named export.
export const handle: Handle = (input) => {
  if (input.event.url.pathname.startsWith("/api/")) {
    return input.resolve(input.event);
  }
  return localizedHandle(input);
};
