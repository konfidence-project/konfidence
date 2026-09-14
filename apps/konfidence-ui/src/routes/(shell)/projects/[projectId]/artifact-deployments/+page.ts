import { ArtifactDeploymentsStore } from "$lib/artifact-deployments/store.svelte";
import { LANDSCAPE_PARAM, VECTOR_DEPLOYMENT_PARAM } from "$lib/artifact-deployments/params";
import { getApiClient } from "$lib/konfidence-api/client-instance";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, url }) => {
  const { projectId } = params;
  const landscapeId = url.searchParams.get(LANDSCAPE_PARAM) ?? undefined;
  const vectorDeploymentId = url.searchParams.get(VECTOR_DEPLOYMENT_PARAM) ?? undefined;
  const store = new ArtifactDeploymentsStore(getApiClient());

  void store.refresh(projectId, { landscapeId, vectorDeploymentId });

  return {
    landscapeId,
    projectId,
    store,
    vectorDeploymentId,
  };
};
