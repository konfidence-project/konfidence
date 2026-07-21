import { type ServerLoad, redirect } from "@sveltejs/kit";

export const load: ServerLoad = async ({ locals, url }) => {
  if (locals.user) {
    const returnTo = url.searchParams.get("returnTo") ?? "/landscape";
    redirect(303, returnTo);
  }
};
