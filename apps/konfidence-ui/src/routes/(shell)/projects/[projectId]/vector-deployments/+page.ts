import { LANDSCAPE_PARAM } from "$lib/deployments/params";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, url }) => {
  const { projectId } = params;
  const landscapeId = url.searchParams.get(LANDSCAPE_PARAM) ?? undefined;

  return {
    landscapeId,
    projectId,
  };
};
