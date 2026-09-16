import type { components, paths } from "@konfidence/api-client/schema";
import type { ApiClient } from "$lib/konfidence-api/client";

type Landscape = components["schemas"]["Landscape"];
type Stage = components["schemas"]["Stage"];

const LANDSCAPES_ROUTE = "/v1/projects/{projectId}/landscapes" satisfies keyof paths;
const STAGES_ROUTE = "/v1/projects/{projectId}/stages" satisfies keyof paths;

type LandscapeResource = "landscapes" | "stages";

class LandscapeApiError extends Error {
  readonly status: number;
  readonly resource: LandscapeResource;

  constructor(resource: LandscapeResource, status: number) {
    super(`Unable to load ${resource} (status ${status})`);
    this.name = "LandscapeApiError";
    this.status = status;
    this.resource = resource;
  }
}

interface GetLandscapesOptions {
  client: ApiClient;
  projectId: string;
  signal: AbortSignal;
}

interface GetStagesOptions extends GetLandscapesOptions {
  landscapeId?: string;
}

const getLandscapes = async ({
  client,
  projectId,
  signal,
}: GetLandscapesOptions): Promise<readonly Landscape[]> => {
  const result = await client.GET(LANDSCAPES_ROUTE, {
    params: { path: { projectId } },
    signal,
  });
  if (!result.data) {
    throw new LandscapeApiError("landscapes", result.response.status);
  }
  return result.data.data;
};

const getStages = async ({
  client,
  projectId,
  signal,
  landscapeId,
}: GetStagesOptions): Promise<readonly Stage[]> => {
  const result = await client.GET(STAGES_ROUTE, {
    params: {
      path: { projectId },
      query: landscapeId === undefined ? undefined : { landscapeId },
    },
    signal,
  });
  if (!result.data) {
    throw new LandscapeApiError("stages", result.response.status);
  }
  return result.data.data;
};

export { getLandscapes, getStages, LandscapeApiError };
export type { Landscape, LandscapeResource, Stage };
