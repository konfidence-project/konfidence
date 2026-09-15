import { getApiClient } from "$lib/konfidence-api/client-instance";
import { LandscapeDataStore } from "$lib/landscape/landscape-data.svelte";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => {
  const { projectId } = params;
  const store = new LandscapeDataStore(getApiClient());

  void store.refresh(projectId);

  return {
    projectId,
    store,
  };
};
