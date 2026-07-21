import { type ServerLoad, redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";

export const load: ServerLoad = async ({ locals, url }) => {
  if (!locals.user) {
    const redirectTo = url.pathname + url.search;
    redirect(HTTP_SEE_OTHER, `/login?returnTo=${encodeURIComponent(redirectTo)}`);
  }

  return { user: locals.user };
};
