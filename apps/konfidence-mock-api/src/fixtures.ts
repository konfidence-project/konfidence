import type { components } from "./generated/schema.js";

type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
type Identity = components["schemas"]["Identity"];
type Landscape = components["schemas"]["Landscape"];
type Project = components["schemas"]["Project"];
type Stage = components["schemas"]["Stage"];
type VectorDeployment = components["schemas"]["VectorDeployment"];

interface ProjectFixture {
  artifactDeployments: ArtifactDeployment[];
  landscapes: Landscape[];
  project: Project;
  roles: string[];
  stages: Stage[];
  vectorDeployments: VectorDeployment[];
}

interface ScenarioFixture {
  projects: ProjectFixture[];
  resourcesUnavailable?: boolean;
  user: Omit<Identity, "projectRoles">;
}

const paymentsProject = { id: "payments-platform", name: "Payments Platform" } satisfies Project;
const identityProject = { id: "identity-service", name: "Identity Service" } satisfies Project;

const landscapes = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
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

const populatedProject = {
  artifactDeployments,
  landscapes,
  project: paymentsProject,
  roles: ["admin", "dev"],
  stages,
  vectorDeployments,
} satisfies ProjectFixture;

const scenarios = {
  admin: {
    projects: [
      populatedProject,
      {
        artifactDeployments: [],
        landscapes: [],
        project: identityProject,
        roles: ["admin"],
        stages: [],
        vectorDeployments: [],
      },
    ],
    user: {
      email: "alex.admin@example.com",
      familyName: "Admin",
      givenName: "Alex",
      name: "Alex Admin",
    },
  },
  degraded: {
    projects: [populatedProject],
    resourcesUnavailable: true,
    user: {
      email: "riley.operator@example.com",
      familyName: "Operator",
      givenName: "Riley",
      name: "Riley Operator",
    },
  },
  developer: {
    projects: [
      {
        artifactDeployments: [],
        landscapes: landscapes.filter(({ id }) => id !== "production"),
        project: paymentsProject,
        roles: ["dev"],
        stages: stages.slice(0, 1),
        vectorDeployments: vectorDeployments.slice(0, 1),
      },
    ],
    user: {
      email: "dana.developer@example.com",
      familyName: "Developer",
      givenName: "Dana",
      name: "Dana Developer",
    },
  },
} satisfies Record<string, ScenarioFixture>;

export { scenarios };
export type { ProjectFixture, ScenarioFixture };
