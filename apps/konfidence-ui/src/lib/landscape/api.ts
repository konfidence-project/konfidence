import type { components, paths } from "@konfidence/api-client/schema";
import type { ApiClient } from "$lib/konfidence-api/client";

type Landscape = components["schemas"]["Landscape"];
type Stage = components["schemas"]["Stage"];

const LANDSCAPES_ROUTE = "/v1/projects/{projectId}/landscapes" satisfies keyof paths;
const STAGES_ROUTE = "/v1/projects/{projectId}/stages" satisfies keyof paths;

interface LandscapeData {
  landscapes: readonly Landscape[];
  stages: readonly Stage[];
}

class LandscapeApiError extends Error {
  readonly status: number;

  constructor(resource: string, status: number) {
    super(`Unable to load ${resource} (status ${status})`);
    this.name = "LandscapeApiError";
    this.status = status;
  }
}

interface GetLandscapeDataOptions {
  client: ApiClient;
  projectId: string;
  signal: AbortSignal;
  landscapeId?: string;
}

const getLandscapeData = async ({
  client,
  projectId,
  signal,
  landscapeId,
}: GetLandscapeDataOptions): Promise<LandscapeData> => {
  const path = { projectId };
  const [landscapes, stages] = await Promise.all([
    client.GET(LANDSCAPES_ROUTE, { params: { path }, signal }),
    client.GET(STAGES_ROUTE, {
      params: {
        path,
        query: landscapeId === undefined ? undefined : { landscapeId },
      },
      signal,
    }),
  ]);

  if (!landscapes.data) {
    throw new LandscapeApiError("landscapes", landscapes.response.status);
  }
  if (!stages.data) {
    throw new LandscapeApiError("stages", stages.response.status);
  }
  return { landscapes: landscapes.data.data, stages: stages.data.data };
};

export { getLandscapeData, LandscapeApiError };
export type { Landscape, LandscapeData, Stage };
