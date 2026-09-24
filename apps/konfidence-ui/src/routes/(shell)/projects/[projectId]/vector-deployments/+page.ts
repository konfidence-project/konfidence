import { VectorDeploymentsStore } from "$lib/vector-deployments/store.svelte";
import { LANDSCAPE_PARAM } from "$lib/vector-deployments/params";
import { getApiClient } from "$lib/konfidence-api/client-instance";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, url }) => {
  const { projectId } = params;
  const landscapeId = url.searchParams.get(LANDSCAPE_PARAM) ?? undefined;
  const store = new VectorDeploymentsStore(getApiClient());

  void store.refresh(projectId, { landscapeId });

  return {
    landscapeId,
    projectId,
    store,
  };
};
