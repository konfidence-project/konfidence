import { getVectorDeployments } from "$lib/konfidence-api/queries.remote";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ params }) => ({
  vectorDeployments: await getVectorDeployments(params.projectId),
});
