import { redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = ({ params }) => {
  redirect(HTTP_SEE_OTHER, `/projects/${params.projectId}/landscape`);
};
