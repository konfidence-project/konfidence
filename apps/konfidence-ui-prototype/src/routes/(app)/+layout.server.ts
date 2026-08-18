import { redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import type { LayoutServerLoad } from "./$types";
import { getProjects } from "$lib/konfidence-api/queries.remote";

export const load: LayoutServerLoad = async ({ locals, url }) => {
  if (!locals.user) {
    const redirectTo = url.pathname + url.search;
    redirect(HTTP_SEE_OTHER, `/api/login?returnTo=${encodeURIComponent(redirectTo)}`);
  }

  return { projects: await getProjects(), user: locals.user };
};
