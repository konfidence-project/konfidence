import { LANDSCAPE_PARAM, VECTOR_DEPLOYMENT_PARAM } from "$lib/deployments/params";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, url }) => {
  const { projectId } = params;
  const landscapeId = url.searchParams.get(LANDSCAPE_PARAM) ?? undefined;
  const vectorDeploymentId = url.searchParams.get(VECTOR_DEPLOYMENT_PARAM) ?? undefined;

  return {
    landscapeId,
    projectId,
    vectorDeploymentId,
  };
};
