import { toVectorDeployments } from "$lib/deployments";
import { getVectorDeployments } from "$lib/konfidence-api/queries.remote";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ params }) => {
  const response = await getVectorDeployments(params.projectId);
  return {
    vectorDeployments: toVectorDeployments(
      response.landscapes,
      response.stages,
      response.vectorDeployments,
    ),
  };
};
