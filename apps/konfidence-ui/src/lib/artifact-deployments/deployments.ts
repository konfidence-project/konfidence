import type { components } from "@konfidence/api-client/schema";

type ApiLandscape = components["schemas"]["Landscape"];
type ApiStage = components["schemas"]["Stage"];
type ApiVectorDeployment = components["schemas"]["VectorDeployment"];
type ApiArtifactDeployment = components["schemas"]["ArtifactDeployment"];

interface ArtifactDeploymentRow {
  id: string;
  component: string;
  version: string;
  repository: string;
  landscape: string;
  landscapeId: string;
  stageNames: string[];
  stageIds: string[];
  vectorDeploymentLabels: string[];
  vectorDeploymentIds: string[];
  status: ApiArtifactDeployment["status"];
  raw: ApiArtifactDeployment;
  relatedVectorDeployments: ApiVectorDeployment[];
}

type ArtifactDeploymentStatus = ArtifactDeploymentRow["status"];

const namesById = <Item extends { id: string; name: string }>(
  items: readonly Item[],
): Map<string, string> => new Map(items.map((item) => [item.id, item.name]));

const vectorLabel = (deployment: ApiVectorDeployment): string =>
  `${deployment.vector.componentName}@${deployment.vector.componentVersion}`;

interface ToArtifactDeploymentRowsInput {
  artifactDeployments: readonly ApiArtifactDeployment[];
  landscapes: readonly ApiLandscape[];
  stages: readonly ApiStage[];
  vectorDeployments: readonly ApiVectorDeployment[];
}

const toArtifactDeploymentRows = ({
  artifactDeployments,
  landscapes,
  stages,
  vectorDeployments,
}: ToArtifactDeploymentRowsInput): ArtifactDeploymentRow[] => {
  const landscapeNames = namesById(landscapes);
  const stageNames = namesById(stages);
  const vectorsById = new Map(vectorDeployments.map((deployment) => [deployment.id, deployment]));

  return artifactDeployments.map((deployment) => {
    const relatedVectors = deployment.vectorDeploymentIds
      .map((id) => vectorsById.get(id))
      .filter((vector): vector is ApiVectorDeployment => vector !== undefined);

    return {
      component: deployment.artifact.componentName,
      id: deployment.id,
      landscape: landscapeNames.get(deployment.landscapeId) ?? deployment.landscapeId,
      landscapeId: deployment.landscapeId,
      raw: deployment,
      relatedVectorDeployments: relatedVectors,
      repository: deployment.artifact.repository,
      stageIds: [...deployment.stageIds],
      stageNames: deployment.stageIds.map((id) => stageNames.get(id) ?? id),
      status: deployment.status,
      vectorDeploymentIds: [...deployment.vectorDeploymentIds],
      vectorDeploymentLabels: deployment.vectorDeploymentIds.map((id) => {
        const vector = vectorsById.get(id);
        return vector ? vectorLabel(vector) : id;
      }),
      version: deployment.artifact.componentVersion,
    };
  });
};

const STATUS_LABEL: Record<ArtifactDeploymentStatus, string> = {
  ArtifactDeployed: "Deployed",
  ArtifactFetched: "Fetched",
};

const STATUS_TONE: Record<ArtifactDeploymentStatus, string> = {
  ArtifactDeployed: "healthy",
  ArtifactFetched: "deploying",
};

const statusLabel = (status: ArtifactDeploymentStatus): string => STATUS_LABEL[status];
const statusTone = (status: ArtifactDeploymentStatus): string => STATUS_TONE[status];

const matchesQuery = (row: ArtifactDeploymentRow, query: string): boolean => {
  const trimmed = query.trim().toLowerCase();
  if (!trimmed) {
    return true;
  }
  const haystack = [
    row.id,
    row.component,
    row.version,
    row.repository,
    row.landscape,
    ...row.stageNames,
    ...row.vectorDeploymentLabels,
    ...row.vectorDeploymentIds,
    statusLabel(row.status),
  ]
    .join(" ")
    .toLowerCase();
  return haystack.includes(trimmed);
};

const matchesStatus = (
  row: ArtifactDeploymentRow,
  status: ArtifactDeploymentStatus | "",
): boolean => status === "" || row.status === status;

export { matchesQuery, matchesStatus, statusLabel, statusTone, toArtifactDeploymentRows };
export type { ArtifactDeploymentRow, ArtifactDeploymentStatus };
