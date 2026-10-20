import { getApiClient } from "$lib/konfidence-api/client-instance";
import type { Stage } from "$lib/konfidence-api/types";

type StageReference = Readonly<{
  projectId: string;
  landscapeId: string;
  stageId: string;
}>;

type GetStageResult = { data: Readonly<Stage> } | { error: true; status: number };

const getStage = async (
  { projectId, landscapeId, stageId }: StageReference,
  signal: AbortSignal,
): Promise<GetStageResult> => {
  const result = await getApiClient().GET(
    "/v1/projects/{projectId}/landscapes/{landscapeId}/stages/{stageId}",
    {
      params: { path: { landscapeId, projectId, stageId } },
      signal,
    },
  );

  if ("error" in result || !result.response.ok || !result.data) {
    return { error: true, status: result.response.status };
  }
  return { data: result.data };
};

export { getStage };
export type { GetStageResult, StageReference };
