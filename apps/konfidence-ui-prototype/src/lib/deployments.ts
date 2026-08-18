import type { components } from "$lib/konfidence-api/schema";

type ApiLandscape = components["schemas"]["LandscapeResponse"];
type ApiStage = components["schemas"]["StageResponse"];
type ApiVectorDeployment = components["schemas"]["VectorDeploymentResponse"];
type ApiArtifactDeployment = components["schemas"]["ArtifactDeploymentResponse"];

interface VectorDeployment {
  component: string;
  id: string;
  landscape: string;
  repository: string;
  stage: string;
  status: ApiVectorDeployment["status"];
  version: string;
}

interface ArtifactDeployment {
  component: string;
  id: string;
  landscape: string;
  repository: string;
  stages: string[];
  status: ApiArtifactDeployment["status"];
  vectorDeployments: string[];
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

const toArtifactDeployments = (
  landscapes: ApiLandscape[],
  stages: ApiStage[],
  deployments: ApiArtifactDeployment[],
): ArtifactDeployment[] => {
  const landscapeNames = namesById(landscapes);
  const stageNames = namesById(stages);
  return deployments.map((deployment) => ({
    component: deployment.artifact.componentName,
    id: deployment.id,
    landscape: landscapeNames.get(deployment.landscapeId) ?? deployment.landscapeId,
    repository: deployment.artifact.repository,
    stages: deployment.stageIds.map((id) => stageNames.get(id) ?? id),
    status: deployment.status,
    vectorDeployments: deployment.vectorDeploymentIds,
    version: deployment.artifact.componentVersion,
  }));
};

export { toArtifactDeployments, toVectorDeployments };
export type { ArtifactDeployment, VectorDeployment };
