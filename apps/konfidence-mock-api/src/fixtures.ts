import type { components } from "./generated/schema.js";

type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
type Identity = components["schemas"]["Identity"];
type Landscape = components["schemas"]["Landscape"];
type Project = components["schemas"]["Project"];
type Stage = components["schemas"]["Stage"];
type StageVersion = components["schemas"]["StageVersion"];
type VectorDeployment = components["schemas"]["VectorDeployment"];
type VectorPromotionConfig = components["schemas"]["VectorPromotionConfig"];

interface ProjectFixture {
  artifactDeployments: ArtifactDeployment[];
  landscapes: Landscape[];
  project: Project;
  roles: string[];
  stages: Stage[];
  vectorDeployments: VectorDeployment[];
  vectorPromotionConfigs: VectorPromotionConfig[];
}

interface ScenarioFixture {
  projects: ProjectFixture[];
  resourcesUnavailable?: boolean;
  user: Omit<Identity, "projectRoles">;
}

const REPOSITORY = "ghcr.io/konfidence/mock";

const paymentsProject = { id: "payments-platform", name: "Payments Platform" } satisfies Project;
const identityProject = { id: "identity-service", name: "Identity Service" } satisfies Project;

// One row per landscape, each with the single stage it holds and the vector deployed there.
const rows = [
  {
    generation: 5,
    landscape: "development",
    landscapeName: "Development",
    stage: "dev-us30",
    vectorStatus: "DeploymentReady",
    version: "2026.8.5",
  },
  {
    generation: 4,
    landscape: "test",
    landscapeName: "Test",
    stage: "test-eu20",
    vectorStatus: "DeploymentReady",
    version: "2026.8.4",
  },
  {
    generation: 3,
    landscape: "production",
    landscapeName: "Production",
    stage: "prod-eu30",
    vectorStatus: "DeployingVector",
    version: "2026.8.3",
  },
] as const;

const landscapes: Landscape[] = rows.map((row) => ({ id: row.landscape, name: row.landscapeName }));

const stages: Stage[] = rows.map((row) => {
  // CRD Stage has no displayName yet, so name mirrors id (metadata.name).
  const version: StageVersion = {
    id: `${row.stage}-v${row.generation}`,
    stageGeneration: row.generation,
    status: "Ready",
    vector: `${REPOSITORY}//delivery-vector:${row.version}`,
  };
  return {
    activeStageVersion: version,
    id: row.stage,
    landscapeId: row.landscape,
    name: row.stage,
    targetStageVersion: version,
  };
});

const vectorDeployments: VectorDeployment[] = rows.map((row) => ({
  id: `vector-${row.stage}-1`,
  landscapeId: row.landscape,
  stageId: row.stage,
  status: row.vectorStatus,
  vector: {
    componentName: "delivery-vector",
    componentVersion: row.version,
    repository: REPOSITORY,
  },
}));

const component = (componentName: string, componentVersion: string) => ({
  componentName,
  componentVersion,
  repository: REPOSITORY,
});

const artifactDeployments: ArtifactDeployment[] = [
  {
    artifact: component("payments-api", "3.4.1"),
    id: "artifact-dev-us30-1",
    landscapeId: "development",
    stageIds: ["dev-us30"],
    status: "ArtifactDeployed",
    vectorDeploymentIds: ["vector-dev-us30-1"],
  },
  {
    artifact: component("payments-ui", "2.7.0"),
    id: "artifact-dev-us30-2",
    landscapeId: "development",
    stageIds: ["dev-us30"],
    status: "ArtifactFetched",
    vectorDeploymentIds: ["vector-dev-us30-1"],
  },
  {
    artifact: component("payments-api", "3.4.0"),
    id: "artifact-test-eu20-1",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "ArtifactDeployed",
    vectorDeploymentIds: ["vector-test-eu20-1"],
  },
];

const vectorPromotionConfigs: VectorPromotionConfig[] = [
  {
    id: "delivery-vector-dev-to-test",
    keepLastPromotions: 5,
    // Ordered newest-first (descending sequence) to mirror the real API.
    promotions: [
      {
        id: "promo-dev-to-test-2",
        requireApproval: true,
        sequence: 2,
        source: { kind: "Stage", landscape: "development", name: "dev-us30" },
        status: "Waiting",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.5`,
      },
      {
        approval: {
          approvedAt: "2026-08-28T09:15:00Z",
          approvedBy: "alex.admin@example.com",
        },
        id: "promo-dev-to-test-1",
        requireApproval: true,
        sequence: 1,
        source: { kind: "Stage", landscape: "development", name: "dev-us30" },
        status: "Succeeded",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.4`,
      },
    ],
    source: { kind: "Stage", landscape: "development", name: "dev-us30" },
    target: { kind: "Stage", landscape: "test", name: "test-eu20" },
    ttlAfterFinished: "168h0m0s",
  },
  {
    id: "delivery-vector-test-to-prod",
    keepLastPromotions: 3,
    promotions: [
      {
        id: "promo-test-to-prod-1",
        requireApproval: false,
        sequence: 1,
        source: { kind: "Stage", landscape: "test", name: "test-eu20" },
        status: "Ready",
        target: { kind: "Stage", landscape: "production", name: "prod-eu30" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.3`,
      },
    ],
    source: { kind: "Stage", landscape: "test", name: "test-eu20" },
    target: { kind: "Stage", landscape: "production", name: "prod-eu30" },
  },
];

const populatedProject: ProjectFixture = {
  artifactDeployments,
  landscapes,
  project: paymentsProject,
  roles: ["admin", "dev"],
  stages,
  vectorDeployments,
  vectorPromotionConfigs,
};

const emptyProject = (project: Project, roles: string[]): ProjectFixture => ({
  artifactDeployments: [],
  landscapes: [],
  project,
  roles,
  stages: [],
  vectorDeployments: [],
  vectorPromotionConfigs: [],
});

const scenarios = {
  // A multi-project administrator: one populated project plus one that has nothing yet.
  admin: {
    projects: [populatedProject, emptyProject(identityProject, ["admin"])],
    user: {
      email: "alex.admin@example.com",
      familyName: "Admin",
      givenName: "Alex",
      name: "Alex Admin",
    },
  },
  // An authenticated operator for whom every project resource request fails.
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
  // A developer with a single sparse project and no access to production.
  developer: {
    projects: [
      {
        ...emptyProject(paymentsProject, ["dev"]),
        landscapes: landscapes.filter(({ id }) => id !== "production"),
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
