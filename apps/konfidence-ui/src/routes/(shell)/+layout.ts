import { getApiClient } from "$lib/konfidence-api/client-instance";
import type { LayoutLoad } from "./$types";
import type { paths } from "$lib/konfidence-api/schema";

const PROJECTS_ROUTE = "/v1/projects" satisfies keyof paths;

/**
 * Load the projects the signed-in user can see. Runs client-side only (the
 * app is `ssr = false`), so we can share the singleton API client and rely
 * on the session cookie already being attached by `credentials: same-origin`.
 *
 * Failures return an empty list. The layout renders a "no projects" state
 * rather than crashing; the auth layer above has already gated us.
 */
export const load: LayoutLoad = async () => {
  const result = await getApiClient().GET(PROJECTS_ROUTE);
  return {
    projects: result.data?.data ?? [],
  };
};
