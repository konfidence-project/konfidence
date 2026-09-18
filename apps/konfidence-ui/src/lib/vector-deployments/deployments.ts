import type { components } from "@konfidence/api-client/schema";
import type { StatusTone } from "$lib/components/table-cells/status-cell.types.js";

type ApiLandscape = components["schemas"]["Landscape"];
type ApiStage = components["schemas"]["Stage"];
type ApiVectorDeployment = components["schemas"]["VectorDeployment"];
type ApiArtifactDeployment = components["schemas"]["ArtifactDeployment"];

interface VectorDeploymentRow {
  id: string;
  component: string;
  version: string;
  repository: string;
  landscape: string;
  landscapeId: string;
  stageName: string;
  stageId: string;
  status: ApiVectorDeployment["status"];
  raw: ApiVectorDeployment;
  relatedArtifactDeployments: ApiArtifactDeployment[];
}

type VectorDeploymentStatus = VectorDeploymentRow["status"];

const namesById = <Item extends { id: string; name: string }>(
  items: readonly Item[],
): Map<string, string> => new Map(items.map((item) => [item.id, item.name]));

interface ToVectorDeploymentRowsInput {
  vectorDeployments: readonly ApiVectorDeployment[];
  landscapes: readonly ApiLandscape[];
  stages: readonly ApiStage[];
  artifactDeployments: readonly ApiArtifactDeployment[];
}

const toVectorDeploymentRows = ({
  vectorDeployments,
  landscapes,
  stages,
  artifactDeployments,
}: ToVectorDeploymentRowsInput): VectorDeploymentRow[] => {
  const landscapeNames = namesById(landscapes);
  const stageNames = namesById(stages);

  return vectorDeployments.map((deployment) => {
    const relatedArtifacts = artifactDeployments.filter((artifact) =>
      new Set(artifact.vectorDeploymentIds).has(deployment.id),
    );

    return {
      component: deployment.vector.componentName,
      id: deployment.id,
      landscape: landscapeNames.get(deployment.landscapeId) ?? deployment.landscapeId,
      landscapeId: deployment.landscapeId,
      raw: deployment,
      relatedArtifactDeployments: relatedArtifacts,
      repository: deployment.vector.repository,
      stageId: deployment.stageId,
      stageName: stageNames.get(deployment.stageId) ?? deployment.stageId,
      status: deployment.status,
      version: deployment.vector.componentVersion,
    };
  });
};

const STATUS_LABEL: Record<VectorDeploymentStatus, string> = {
  DeployingVector: "Deploying",
  DeploymentFailed: "Failed",
  DeploymentReady: "Ready",
};

const STATUS_TONE: Record<VectorDeploymentStatus, StatusTone> = {
  DeployingVector: "deploying",
  DeploymentFailed: "error",
  DeploymentReady: "healthy",
};

const statusLabel = (status: VectorDeploymentStatus): string => STATUS_LABEL[status];
const statusTone = (status: VectorDeploymentStatus): StatusTone => STATUS_TONE[status];

const matchesQuery = (row: VectorDeploymentRow, query: string): boolean => {
  const trimmed = query.trim().toLowerCase();
  if (!trimmed) {
    return true;
  }
  const searchableText = [
    row.id,
    row.component,
    row.version,
    row.repository,
    row.landscape,
    row.stageName,
    statusLabel(row.status),
  ]
    .join(" ")
    .toLowerCase();
  return searchableText.includes(trimmed);
};

const matchesStatus = (row: VectorDeploymentRow, status: VectorDeploymentStatus | ""): boolean =>
  status === "" || row.status === status;

export { matchesQuery, matchesStatus, statusLabel, statusTone, toVectorDeploymentRows };
export type { VectorDeploymentRow, VectorDeploymentStatus };
