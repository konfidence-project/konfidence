import type { components } from "$lib/konfidence-api/schema";

type ApiLandscape = components["schemas"]["LandscapeResponse"];
type ApiStage = components["schemas"]["StageResponse"];
type ApiVectorDeployment = components["schemas"]["VectorDeploymentResponse"];

interface VectorDeployment {
  component: string;
  id: string;
  landscape: string;
  repository: string;
  stage: string;
  status: ApiVectorDeployment["status"];
  version: string;
}

const namesById = (items: { id: string; name: string }[]): Map<string, string> =>
  new Map(items.map((item) => [item.id, item.name]));

const toVectorDeployments = (
  landscapes: ApiLandscape[],
  stages: ApiStage[],
  deployments: ApiVectorDeployment[],
): VectorDeployment[] => {
  const landscapeNames = namesById(landscapes);
  const stageNames = namesById(stages);
  return deployments.map((deployment) => ({
    component: deployment.vector.componentName,
    id: deployment.id,
    landscape: landscapeNames.get(deployment.landscapeId) ?? deployment.landscapeId,
    repository: deployment.vector.repository,
    stage: stageNames.get(deployment.stageId) ?? deployment.stageId,
    status: deployment.status,
    version: deployment.vector.componentVersion,
  }));
};

export { toVectorDeployments };
export type { VectorDeployment };
