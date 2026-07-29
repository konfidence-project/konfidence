import { error } from "@sveltejs/kit";
import { HTTP_NOT_FOUND } from "$lib/http-status";
import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = async ({ params, parent }) => {
  const { projects } = await parent();
  const project = projects.find((candidate) => candidate.id === params.projectId);
  if (!project) {
    error(HTTP_NOT_FOUND, "Project not found");
  }
  return { project };
};
