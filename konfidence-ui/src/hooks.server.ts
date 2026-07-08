import { SESSION_COOKIE_NAME, getSession } from "$lib/server/session";
import type { Handle } from "@sveltejs/kit";

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit requires a named `handle` export.
export const handle: Handle = async ({ event, resolve }) => {
  const sid = event.cookies.get(SESSION_COOKIE_NAME);
  event.locals.user = getSession(sid);
  return resolve(event);
};
