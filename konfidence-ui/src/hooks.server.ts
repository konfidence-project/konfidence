import type { Handle } from "@sveltejs/kit";
import { resolveSession } from "$lib/server/auth";

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit expects this named export.
export const handle: Handle = async ({ event, resolve }) => {
  if (event.route.id === "/api/[...path]") {
    return resolve(event);
  }

  const session = await resolveSession(event);
  event.locals.session = session;
  event.locals.user = session?.user;

  return resolve(event);
};
