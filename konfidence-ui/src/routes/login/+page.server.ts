import { type ServerLoad, redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";

export const load: ServerLoad = async ({ locals, url }) => {
  if (locals.user) {
    const returnTo = url.searchParams.get("returnTo") ?? "/landscape";
    redirect(HTTP_SEE_OTHER, returnTo);
  }
};
