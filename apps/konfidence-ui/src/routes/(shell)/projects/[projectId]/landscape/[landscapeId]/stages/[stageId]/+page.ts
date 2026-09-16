import { getApiClient } from "$lib/konfidence-api/client-instance";
import { LandscapeDataStore } from "$lib/landscape/landscapeData.svelte";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => {
  const { projectId, landscapeId, stageId } = params;
  const store = new LandscapeDataStore(getApiClient());

  void store.refresh(projectId, { landscapeId });

  return {
    landscapeId,
    projectId,
    stageId,
    store,
  };
};
