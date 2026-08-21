import type { components } from "./generated/schema.js";

type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
type Landscape = components["schemas"]["Landscape"];
type Project = components["schemas"]["Project"];
type Stage = components["schemas"]["Stage"];
type VectorDeployment = components["schemas"]["VectorDeployment"];

const projects = [
  { id: "payments-platform", name: "Payments Platform" },
  { id: "identity-service", name: "Identity Service" },
  { id: "analytics-pipeline", name: "Analytics Pipeline" },
  { id: "legacy-migration", name: "Legacy Migration" },
] satisfies Project[];

const landscapes = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
  { id: "performance", name: "Performance" },
  { id: "production", name: "Production" },
] satisfies Landscape[];

const stages = [
  {
    id: "dev-us30",
    landscapeId: "development",
    name: "Development US",
    targetStageVersion: {
      active: true,
      id: "dev-us30-v5",
      stageGeneration: 5,
      status: "DeploymentCreated",
      vector: "ghcr.io/konfidence/mock//delivery-vector:2026.8.5",
    },
  },
  {
    id: "test-eu20",
    landscapeId: "test",
    name: "Test EU",
    targetStageVersion: {
      active: true,
      id: "test-eu20-v4",
      stageGeneration: 4,
      status: "DeploymentCreated",
      vector: "ghcr.io/konfidence/mock//delivery-vector:2026.8.4",
    },
  },
  {
    id: "prod-eu30",
    landscapeId: "production",
    name: "Production EU",
    targetStageVersion: {
      active: true,
      id: "prod-eu30-v3",
      stageGeneration: 3,
      status: "DeploymentCreated",
      vector: "ghcr.io/konfidence/mock//delivery-vector:2026.8.3",
    },
  },
] satisfies Stage[];

const vectorDeployments = [
  {
    id: "vector-dev-us30-1",
    landscapeId: "development",
    stageId: "dev-us30",
    status: "ArtifactDeploymentCreated",
    vector: {
      componentName: "delivery-vector",
      componentVersion: "2026.8.5",
      repository: "ghcr.io/konfidence/mock",
    },
  },
  {
    id: "vector-test-eu20-1",
    landscapeId: "test",
    stageId: "test-eu20",
    status: "ArtifactDeploymentCreated",
    vector: {
      componentName: "delivery-vector",
      componentVersion: "2026.8.4",
      repository: "ghcr.io/konfidence/mock",
    },
  },
  {
    id: "vector-prod-eu30-1",
    landscapeId: "production",
    stageId: "prod-eu30",
    status: "VectorDownloaded",
    vector: {
      componentName: "delivery-vector",
      componentVersion: "2026.8.3",
      repository: "ghcr.io/konfidence/mock",
    },
  },
] satisfies VectorDeployment[];

const artifactDeployments = [
  {
    artifact: {
      componentName: "payments-api",
      componentVersion: "3.4.1",
      repository: "ghcr.io/konfidence/mock",
    },
    id: "artifact-dev-us30-1",
    landscapeId: "development",
    stageIds: ["dev-us30"],
    status: "ArtifactDeployed",
    vectorDeploymentIds: ["vector-dev-us30-1"],
  },
  {
    artifact: {
      componentName: "payments-ui",
      componentVersion: "2.7.0",
      repository: "ghcr.io/konfidence/mock",
    },
    id: "artifact-dev-us30-2",
    landscapeId: "development",
    stageIds: ["dev-us30"],
    status: "ArtifactFetched",
    vectorDeploymentIds: ["vector-dev-us30-1"],
  },
  {
    artifact: {
      componentName: "payments-api",
      componentVersion: "3.4.0",
      repository: "ghcr.io/konfidence/mock",
    },
    id: "artifact-test-eu20-1",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "ArtifactDeployed",
    vectorDeploymentIds: ["vector-test-eu20-1"],
  },
] satisfies ArtifactDeployment[];

const forEachProject = <Item>(items: Item[]): Record<string, Item[]> =>
  Object.fromEntries(projects.map((project) => [project.id, items]));

const fixtures = {
  artifactDeploymentsByProject: forEachProject(artifactDeployments),
  landscapesByProject: forEachProject(landscapes),
  projects,
  stagesByProject: forEachProject(stages),
  vectorDeploymentsByProject: forEachProject(vectorDeployments),
};

export { fixtures };
