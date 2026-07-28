import { requireUser } from "$lib/server/auth";
import type { LayoutServerLoad } from "./$types";
import { getProjects } from "$lib/konfidence-api/queries.remote";

export const load: LayoutServerLoad = async ({ locals }) => {
  const user = requireUser(locals);
  return { projects: await getProjects(), user };
};
