import { redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ parent }) => {
  const { projects } = await parent();
  if (projects.length === 1) {
    redirect(HTTP_SEE_OTHER, `/projects/${projects[0].id}/landscape`);
  }
  redirect(HTTP_SEE_OTHER, "/projects");
};
