import { type ServerLoad, redirect } from "@sveltejs/kit";

const HTTP_SEE_OTHER = 303;

export const load: ServerLoad = async ({ locals, url }) => {
  if (!locals.user) {
    const redirectTo = url.pathname + url.search;
    redirect(HTTP_SEE_OTHER, `/api/login?returnTo=${encodeURIComponent(redirectTo)}`);
  }

  return { user: locals.user };
};
