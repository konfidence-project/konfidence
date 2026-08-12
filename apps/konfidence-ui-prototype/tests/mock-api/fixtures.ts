import type { components } from "../../src/lib/konfidence-api/schema";

type Project = components["schemas"]["Project"];
type Landscape = components["schemas"]["Landscape"];
type Stage = components["schemas"]["Stage"];
type StageStatus = Stage["targetStageVersion"]["status"];
type VectorDeployment = components["schemas"]["VectorDeployment"];
type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];

interface StageDefinition {
  id: string;
  landscapeId: string;
  status: StageStatus;
  vectorCount: number;
}

const HEX_RADIX = 16;
const REVISION_WIDTH = 7;
const ARTIFACT_COUNT_MULTIPLIER = 2;
const ARTIFACT_COUNT_MODULUS = 7;
const EVEN_DIVISOR = 2;

const projects = [
  { id: "payments-platform", name: "Payments Platform" },
  { id: "identity-service", name: "Identity Service" },
  { id: "analytics-pipeline", name: "Analytics Pipeline" },
  { id: "legacy-migration", name: "Legacy Migration" },
] satisfies Project[];

const deliveryLandscapes = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
  { id: "performance", name: "Performance" },
  { id: "production", name: "Production" },
] satisfies Landscape[];

const deliveryStageDefinitions = [
  { id: "dev-us30", landscapeId: "development", status: "DeploymentCreated", vectorCount: 5 },
  { id: "dev-eu12", landscapeId: "development", status: "DeploymentCreated", vectorCount: 3 },
  { id: "test-us20", landscapeId: "test", status: "DeploymentCreated", vectorCount: 4 },
  { id: "perf-us30", landscapeId: "performance", status: "DeploymentCreated", vectorCount: 2 },
  { id: "prod-eu30", landscapeId: "production", status: "DeploymentCreated", vectorCount: 2 },
  { id: "prod-us30", landscapeId: "production", status: "DeploymentCreated", vectorCount: 2 },
  { id: "prod-ap30", landscapeId: "production", status: "DeploymentCreated", vectorCount: 1 },
  { id: "tsm-prod", landscapeId: "production", status: "DeploymentCreated", vectorCount: 1 },
] satisfies StageDefinition[];

const deliveryStages = deliveryStageDefinitions.map(
  (definition, index): Stage => ({
    id: definition.id,
    landscapeId: definition.landscapeId,
    name: definition.id,
    targetStageVersion: {
      active: false,
      id: `${definition.id}-v${index + 1}`,
      stageGeneration: index + 1,
      status: definition.status,
      vector: `2026.${index + 1}.0-${(index + 1).toString(HEX_RADIX).padStart(REVISION_WIDTH, "0")}`,
    },
  }),
);

const artifactCountFor = (stageIndex: number, vectorIndex: number): number =>
  ((stageIndex * ARTIFACT_COUNT_MULTIPLIER + vectorIndex) % ARTIFACT_COUNT_MODULUS) + 1;

const artifactStatusFor = (artifactIndex: number): ArtifactDeployment["status"] => {
  if (artifactIndex % EVEN_DIVISOR === 0) {
    return "ArtifactFetched";
  }
  return "ArtifactDeployed";
};

const createDeliveryDeployments = (): {
  artifacts: ArtifactDeployment[];
  vectors: VectorDeployment[];
} => {
  const artifacts: ArtifactDeployment[] = [];
  const vectors: VectorDeployment[] = [];

  deliveryStageDefinitions.forEach((stage, stageIndex) => {
    for (let vectorIndex = 1; vectorIndex <= stage.vectorCount; vectorIndex += 1) {
      const vectorId = `vector-${stage.id}-${vectorIndex}`;
      const componentName = `${stage.id}-vector-${vectorIndex}`;
      const componentVersion = `2026.${stageIndex + 1}.${vectorIndex}`;
      vectors.push({
        id: vectorId,
        landscapeId: stage.landscapeId,
        stageId: stage.id,
        status: "ArtifactDeploymentCreated",
        vector: {
          componentName,
          componentVersion,
          repository: "ghcr.io/konfidence/mock",
        },
      });

      const artifactCount = artifactCountFor(stageIndex, vectorIndex);
      for (let artifactIndex = 1; artifactIndex <= artifactCount; artifactIndex += 1) {
        artifacts.push({
          artifact: {
            componentName: `${componentName}-artifact-${artifactIndex}`,
            componentVersion: `${componentVersion}.${artifactIndex}`,
            repository: "ghcr.io/konfidence/mock",
          },
          id: `artifact-${stage.id}-${vectorIndex}-${artifactIndex}`,
          landscapeId: stage.landscapeId,
          stageIds: [stage.id],
          status: artifactStatusFor(artifactIndex),
          vectorDeploymentIds: [vectorId],
        });
      }
    }
  });

  return { artifacts, vectors };
};

const deliveryDeployments = createDeliveryDeployments();

const forEachProject = <Item>(items: Item[]): Record<string, Item[]> =>
  Object.fromEntries(projects.map((project) => [project.id, items]));

const landscapesByProject = forEachProject(deliveryLandscapes);
const stagesByProject = forEachProject(deliveryStages);
const vectorDeploymentsByProject = forEachProject(deliveryDeployments.vectors);
const artifactDeploymentsByProject = forEachProject(deliveryDeployments.artifacts);

export {
  artifactDeploymentsByProject,
  landscapesByProject,
  projects,
  stagesByProject,
  vectorDeploymentsByProject,
};
