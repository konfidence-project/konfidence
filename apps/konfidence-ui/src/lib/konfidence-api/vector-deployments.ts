import { getApiClient } from "$lib/konfidence-api/client-instance";
import type { VectorDeployment } from "$lib/konfidence-api/types";

type VectorDeploymentReference = Readonly<{
  projectId: string;
  landscapeId: string;
  vectorDeploymentId: string;
}>;

type GetVectorDeploymentResult =
  | { data: Readonly<VectorDeployment> }
  | { error: true; status: number };

const getVectorDeployment = async (
  { projectId, landscapeId, vectorDeploymentId }: VectorDeploymentReference,
  signal: AbortSignal,
): Promise<GetVectorDeploymentResult> => {
  const result = await getApiClient().GET(
    "/v1/projects/{projectId}/landscapes/{landscapeId}/vectorDeployments/{vectorDeploymentId}",
    {
      params: { path: { landscapeId, projectId, vectorDeploymentId } },
      signal,
    },
  );

  if ("error" in result || !result.response.ok || !result.data) {
    return { error: true, status: result.response.status };
  }
  return { data: result.data };
};

export { getVectorDeployment };
export type { GetVectorDeploymentResult, VectorDeploymentReference };
