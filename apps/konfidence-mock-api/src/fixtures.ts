/* oxlint-disable eslint/no-magic-numbers, eslint/max-params -- Fixture builders keep explicit generations and API field values together. */
import type { components } from "@konfidence/api-client/schema";

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

const paymentsProject = {
  id: "payments-platform",
  name: "Payments Platform",
} satisfies Project;
const identityProject = {
  id: "identity-service",
  name: "Identity Service",
} satisfies Project;

// Vector deployments for the three original stages.
const rows = [
  {
    landscape: "development",
    stage: "dev-us30",
    vectorStatus: "DeploymentReady",
    version: "2026.8.5",
  },
  {
    landscape: "test",
    stage: "test-eu20",
    vectorStatus: "DeploymentReady",
    version: "2026.8.4",
  },
  {
    landscape: "production",
    stage: "prod-eu30",
    vectorStatus: "DeployingVector",
    version: "2026.8.3",
  },
] as const;

const landscapes: Landscape[] = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
  { id: "production", name: "Production" },
  { id: "sandbox", name: "Sandbox" },
];

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

const stageVersion = (
  stageId: string,
  stageGeneration: number,
  status: StageVersion["status"],
  version: string,
): StageVersion => ({
  id: `${stageId}-v${stageGeneration}`,
  stageGeneration,
  status,
  vector: `${REPOSITORY}//delivery-vector:${version}`,
});

const stage = (
  id: string,
  name: string,
  landscapeId: string,
  versions: Pick<Stage, "activeStageVersion" | "targetStageVersion">,
): Stage => ({ id, landscapeId, name, ...versions });

const sandboxVersion = stageVersion("sandbox", 2, "Ready", "2026.8.2");
const devVersion = stageVersion("dev-us30", 5, "Ready", "2026.8.5");
const testVersion = stageVersion("test-eu20", 4, "Ready", "2026.8.4");
const prodVersion = stageVersion("prod-eu30", 3, "Ready", "2026.8.3");

const stages: Stage[] = [
  stage("dev-us30", "dev-us30", "development", {
    activeStageVersion: devVersion,
    targetStageVersion: devVersion,
  }),
  stage("test-eu20", "test-eu20", "test", {
    activeStageVersion: testVersion,
    targetStageVersion: testVersion,
  }),
  stage("prod-eu30", "prod-eu30", "production", {
    activeStageVersion: prodVersion,
    targetStageVersion: prodVersion,
  }),
  stage("dev-eu10", "dev-eu10", "development", {
    activeStageVersion: stageVersion("dev-eu10", 5, "Ready", "2026.8.5"),
    targetStageVersion: stageVersion("dev-eu10", 6, "DeployingVector", "2026.8.6"),
  }),
  stage("dev-canary", "DEV-canary", "development", {
    targetStageVersion: stageVersion("dev-canary", 1, "PendingDeployment", "2026.9.0"),
  }),
  stage("dev-rollback", "dev-rollback", "development", {
    activeStageVersion: stageVersion("dev-rollback", 1, "Ready", "2026.8.1"),
    targetStageVersion: stageVersion("dev-rollback", 2, "Failed", "2026.8.7"),
  }),
  stage("dev-new", "dev-new", "development", {}),
  stage("sandbox", "sandbox", "development", {
    activeStageVersion: sandboxVersion,
    targetStageVersion: sandboxVersion,
  }),
  stage("test-us10", "test-us10", "test", {
    activeStageVersion: stageVersion("test-us10", 4, "Ready", "2026.8.4"),
    targetStageVersion: stageVersion("test-us10", 5, "MigratingVector", "2026.8.5"),
  }),
  stage("test-hotfix", "Test-hotfix", "test", {
    activeStageVersion: stageVersion("test-hotfix", 5, "Ready", "2026.8.5"),
    targetStageVersion: stageVersion("test-hotfix", 6, "ActivatingVector", "2026.8.6"),
  }),
  stage("test-broken", "test-broken", "test", {
    activeStageVersion: stageVersion("test-broken", 2, "Ready", "2026.7.9"),
    targetStageVersion: stageVersion("test-broken", 3, "Failed", "2026.8.0"),
  }),
  stage("test-green", "test-green", "test", {
    activeStageVersion: stageVersion("test-green", 3, "Ready", "2026.8.3"),
    targetStageVersion: stageVersion("test-green", 4, "Ready", "2026.8.4"),
  }),
  stage("prod-us40", "prod-us40", "production", {
    activeStageVersion: stageVersion("prod-us40", 3, "Ready", "2026.8.3"),
    targetStageVersion: stageVersion("prod-us40", 4, "ActivatingVector", "2026.8.4"),
  }),
  stage("prod-dr", "PROD-dr", "production", {
    targetStageVersion: stageVersion("prod-dr", 1, "PendingDeployment", "2026.8.8"),
  }),
  stage("prod-legacy", "prod-legacy", "production", {
    activeStageVersion: stageVersion("prod-legacy", 1, "Ready", "2026.6.0"),
    targetStageVersion: stageVersion("prod-legacy", 2, "Failed", "2026.7.0"),
  }),
];

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
  {
    artifact: component("payments-api", "3.4.1"),
    id: "artifact-test-eu20-2",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "AppHealthy",
    vectorDeploymentIds: ["vector-test-eu20-1"],
  },
  {
    artifact: component("payments-api", "3.4.1"),
    id: "artifact-test-eu20-3",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "Ready",
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
        landscapes: landscapes.filter(({ id }) => id === "development" || id === "test"),
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
