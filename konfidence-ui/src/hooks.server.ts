import { type Handle, redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import { resolveSession } from "$lib/server/auth";
import { THEME_COOKIE, parseTheme } from "$lib/theme";

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit expects this named export.
export const handle: Handle = async ({ event, resolve }) => {
  const startedAt = Date.now();

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
  let themeClass = "";
  if (theme === "konfidence-dark") {
    themeClass = "dark";
  }
  const response = await resolve(event, {
    transformPageChunk: ({ html }) =>
      html.replace("%konfidence.theme%", theme).replace("%konfidence.theme-class%", themeClass),
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
