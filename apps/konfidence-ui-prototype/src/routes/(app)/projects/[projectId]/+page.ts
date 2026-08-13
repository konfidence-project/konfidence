import { redirect } from "@sveltejs/kit";
import { HTTP_SEE_OTHER } from "$lib/http-status";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => {
  redirect(HTTP_SEE_OTHER, `/projects/${params.projectId}/landscape`);
};
