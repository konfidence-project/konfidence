import type { Handle } from "@sveltejs/kit";
import { resolveSession } from "$lib/server/auth";

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit expects this named export.
export const handle: Handle = async ({ event, resolve }) => {
  const startedAt = Date.now();

  if (event.route.id?.startsWith("/(app)") === true) {
    const session = await resolveSession(event);
    event.locals.session = session;
    event.locals.user = session?.user;
  }

  const response = await resolve(event);

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
