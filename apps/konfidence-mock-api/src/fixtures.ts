/* oxlint-disable eslint/no-magic-numbers, eslint/max-params -- Fixture builders keep explicit generations and API field values together. */
import type { components } from "@konfidence/api-client/schema";

type ArtifactContent = components["schemas"]["ArtifactContent"];
type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
type Identity = components["schemas"]["Identity"];
type Landscape = components["schemas"]["Landscape"];
type Project = components["schemas"]["Project"];
type Stage = components["schemas"]["Stage"];
type StageVersion = components["schemas"]["StageVersion"];
type VectorContent = components["schemas"]["VectorContent"];
type VectorDeployment = components["schemas"]["VectorDeployment"];
type VectorPromotionConfig = components["schemas"]["VectorPromotionConfig"];

interface ProjectFixture {
  artifactDeployments: ArtifactDeployment[];
  landscapes: Landscape[];
  project: Project;
  roles: string[];
  stages: Stage[];
  artifactContents: ArtifactContent[];
  vectorContents: VectorContent[];
  // Only the list-page rows: backs listVectorDeploymentsV1.
  vectorDeployments: VectorDeployment[];
  // Superset the single-item fetch resolves: list-page rows plus every stageVersion-derived deployment.
  resolvableVectorDeployments: VectorDeployment[];
  vectorPromotionConfigs: VectorPromotionConfig[];
}

interface ScenarioFixture {
  projects: ProjectFixture[];
  resourcesUnavailable?: boolean;
  user: Omit<Identity, "projectRoles">;
}

const VECTOR_REPOSITORY = "https://bp.common.repositories.cloud.sap/konfidence";
const REPOSITORY = `${VECTOR_REPOSITORY}/delivery-vector`;
const VECTOR_COMPONENT = "github.com/konfidence-project/example-app/vector";
const PAYMENTS_API_COMPONENT = "github.com/konfidence-project/example-app/payments-api";
const PAYMENTS_UI_COMPONENT = "github.com/konfidence-project/example-app/payments-ui";

const VECTOR_DEPLOYMENT_IDS = {
  development: "dev-us30-20260929-201426000Z",
  production: "prod-eu30-20260918-094500000Z",
  test: "test-eu20-20260924-151030000Z",
} as const;

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
    id: VECTOR_DEPLOYMENT_IDS.development,
    landscape: "development",
    stage: "dev-us30",
    vectorStatus: "DeploymentReady",
    version: "2026.9.29-201426000Z",
  },
  {
    id: VECTOR_DEPLOYMENT_IDS.test,
    landscape: "test",
    stage: "test-eu20",
    vectorStatus: "DeploymentReady",
    version: "2026.9.24-151030000Z",
  },
  {
    id: VECTOR_DEPLOYMENT_IDS.production,
    landscape: "production",
    stage: "prod-eu30",
    vectorStatus: "DeployingVector",
    version: "2026.9.18-094500000Z",
  },
] as const;

const landscapes: Landscape[] = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
  { id: "production", name: "Production" },
  { id: "sandbox", name: "Sandbox" },
  // Intentionally stage-less to exercise the empty-landscape placeholder path.
  { id: "staging", name: "Staging" },
];

const vectorDeployments: VectorDeployment[] = rows.map((row) => ({
  id: row.id,
  landscapeId: row.landscape,
  stageId: row.stage,
  status: row.vectorStatus,
  vector: {
    componentName: VECTOR_COMPONENT,
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
  vector: `${VECTOR_REPOSITORY}//delivery-vector:${version}`,
});

const stage = (
  id: string,
  name: string,
  landscapeId: string,
  versions: Pick<Stage, "activeStageVersion" | "targetStageVersion">,
): Stage => ({ id, landscapeId, name, ...versions });

const sandboxVersion = stageVersion("sandbox", 2, "Ready", "2026.9.17-101500000Z");
const devVersion = stageVersion("dev-us30", 5, "Ready", "2026.9.29-201426000Z");
const testVersion = stageVersion("test-eu20", 4, "Ready", "2026.9.24-151030000Z");
const prodVersion = stageVersion("prod-eu30", 3, "Ready", "2026.9.18-094500000Z");

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
    activeStageVersion: stageVersion("dev-eu10", 5, "Ready", "2026.9.23-110000000Z"),
    targetStageVersion: stageVersion("dev-eu10", 6, "DeployingVector", "2026.9.29-201426000Z"),
  }),
  stage("dev-canary", "DEV-canary", "development", {
    targetStageVersion: stageVersion("dev-canary", 1, "PendingDeployment", "2026.9.29-201426000Z"),
  }),
  stage("dev-rollback", "dev-rollback", "development", {
    activeStageVersion: stageVersion("dev-rollback", 1, "Ready", "2026.9.10-083000000Z"),
    targetStageVersion: stageVersion("dev-rollback", 2, "Failed", "2026.9.12-140000000Z"),
  }),
  stage("dev-new", "dev-new", "development", {}),
  stage("sandbox", "sandbox", "development", {
    activeStageVersion: sandboxVersion,
    targetStageVersion: sandboxVersion,
  }),
  stage("test-us10", "test-us10", "test", {
    activeStageVersion: stageVersion("test-us10", 4, "Ready", "2026.9.20-091500000Z"),
    targetStageVersion: stageVersion("test-us10", 5, "MigratingVector", "2026.9.24-151030000Z"),
  }),
  stage("test-hotfix", "Test-hotfix", "test", {
    activeStageVersion: stageVersion("test-hotfix", 5, "Ready", "2026.9.22-120000000Z"),
    targetStageVersion: stageVersion("test-hotfix", 6, "ActivatingVector", "2026.9.24-151030000Z"),
  }),
  stage("test-broken", "test-broken", "test", {
    activeStageVersion: stageVersion("test-broken", 2, "Ready", "2026.8.29-070000000Z"),
    targetStageVersion: stageVersion("test-broken", 3, "Failed", "2026.9.01-070000000Z"),
  }),
  stage("test-green", "test-green", "test", {
    activeStageVersion: stageVersion("test-green", 3, "Ready", "2026.9.18-094500000Z"),
    targetStageVersion: stageVersion("test-green", 4, "Ready", "2026.9.24-151030000Z"),
  }),
  stage("prod-us40", "prod-us40", "production", {
    activeStageVersion: stageVersion("prod-us40", 3, "Ready", "2026.9.18-094500000Z"),
    targetStageVersion: stageVersion("prod-us40", 4, "ActivatingVector", "2026.9.24-151030000Z"),
  }),
  stage("prod-dr", "PROD-dr", "production", {
    targetStageVersion: stageVersion("prod-dr", 1, "PendingDeployment", "2026.9.26-060000000Z"),
  }),
  stage("prod-legacy", "prod-legacy", "production", {
    activeStageVersion: stageVersion("prod-legacy", 1, "Ready", "2026.7.30-060000000Z"),
    targetStageVersion: stageVersion("prod-legacy", 2, "Failed", "2026.8.15-060000000Z"),
  }),
];

// Maps a stageVersion rollout state onto the matching vectorDeployment status.
const vectorDeploymentStatus = (status: StageVersion["status"]): VectorDeployment["status"] => {
  switch (status) {
    case "DeployingVector": {
      return "DeployingVector";
    }
    case "Failed": {
      return "DeploymentFailed";
    }
    // Transient and healthy states both resolve to a ready deployment.
    default: {
      return "DeploymentReady";
    }
  }
};

// Reads the component version out of a stageVersion vector reference tag.
const versionFromVector = (vector: string): string => vector.split(":").at(-1) ?? "";

// Derives one deployment per stageVersion so getVectorDeploymentV1 resolves the stageVersion ids the UI requests (see StageDetails.svelte and useVectorDeployment).
const stageVersionDeployments: VectorDeployment[] = stages.flatMap((entry) =>
  [entry.targetStageVersion, entry.activeStageVersion]
    .filter((version): version is StageVersion => version !== undefined)
    .map((version) => ({
      id: version.id,
      landscapeId: entry.landscapeId,
      stageId: entry.id,
      status: vectorDeploymentStatus(version.status),
      vector: {
        componentName: VECTOR_COMPONENT,
        componentVersion: versionFromVector(version.vector),
        repository: REPOSITORY,
      },
    })),
);

// De-duplicates on id so overlapping active/target versions and list-page fixtures each appear once.
// Only the single-item fetch (getVectorDeploymentV1) draws from this superset; the list page serves vectorDeployments.
const resolvableVectorDeployments: VectorDeployment[] = [
  ...vectorDeployments,
  ...stageVersionDeployments,
].filter((deployment, index, all) => all.findIndex(({ id }) => id === deployment.id) === index);

const vectorContents: VectorContent[] = rows.map((row) => ({
  ref: `${VECTOR_REPOSITORY}//test-vector:${row.version}`,
  components: [
    { componentName: PAYMENTS_API_COMPONENT, version: row.version },
    { componentName: PAYMENTS_UI_COMPONENT, version: row.version },
  ],
}));

const artifactContents: ArtifactContent[] = [
  {
    ref: `${VECTOR_REPOSITORY}//test-artifact/${PAYMENTS_API_COMPONENT}:${rows[0].version}`,
    type: "Helm",
    allowReuse: false,
    tasks: [{ name: "deploy", type: "Helm" }],
    resources: [{ name: "chart", type: "helmChart" }],
  },
  {
    ref: `${VECTOR_REPOSITORY}//test-artifact/${PAYMENTS_UI_COMPONENT}:${rows[0].version}`,
    type: "Helm",
    allowReuse: true,
    tasks: [{ name: "deploy", type: "Helm" }],
    resources: [{ name: "chart", type: "helmChart" }],
  },
];

const component = (componentName: string, componentVersion: string) => ({
  componentName,
  componentVersion,
  repository: REPOSITORY,
});

const artifactDeployments: ArtifactDeployment[] = [
  {
    artifact: component(PAYMENTS_API_COMPONENT, "2026.9.29-201426000Z"),
    id: "dev-us30-20260929-201426000Z-payments-api",
    landscapeId: "development",
    stageIds: ["dev-us30"],
    status: "ArtifactDeployed",
    vectorDeploymentIds: [VECTOR_DEPLOYMENT_IDS.development],
  },
  {
    artifact: component(PAYMENTS_UI_COMPONENT, "2026.9.29-201426000Z"),
    id: "dev-us30-20260929-201426000Z-payments-ui",
    landscapeId: "development",
    stageIds: ["dev-us30"],
    status: "ArtifactFetched",
    vectorDeploymentIds: [VECTOR_DEPLOYMENT_IDS.development],
  },
  {
    artifact: component(PAYMENTS_API_COMPONENT, "2026.9.24-151030000Z"),
    id: "test-eu20-20260924-151030000Z-payments-api-1",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "ArtifactDeployed",
    vectorDeploymentIds: [VECTOR_DEPLOYMENT_IDS.test],
  },
  {
    artifact: component(PAYMENTS_API_COMPONENT, "2026.9.24-151030000Z"),
    id: "test-eu20-20260924-151030000Z-payments-api-2",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "AppHealthy",
    vectorDeploymentIds: [VECTOR_DEPLOYMENT_IDS.test],
  },
  {
    artifact: component(PAYMENTS_API_COMPONENT, "2026.9.24-151030000Z"),
    id: "test-eu20-20260924-151030000Z-payments-api-3",
    landscapeId: "test",
    stageIds: ["test-eu20"],
    status: "Ready",
    vectorDeploymentIds: [VECTOR_DEPLOYMENT_IDS.test],
  },
];

const vectorPromotionConfigs: VectorPromotionConfig[] = [
  {
    gates: [
      { type: "activated-on-source-stage.konfidence.cloud" },
      { type: "explicit-approval.konfidence.cloud" },
    ],
    id: "delivery-vector-dev-to-test",
    keepLastPromotions: 5,
    // Ordered newest-first (descending sequence) to mirror the real API.
    promotions: [
      {
        gates: [
          {
            reason: "ActivatedOnSource",
            setAt: "2026-09-29T20:30:00Z",
            setBy: "konfidence/activated-on-source-stage-controller",
            state: "Satisfied",
            type: "activated-on-source-stage.konfidence.cloud",
          },
          { state: "Pending", type: "explicit-approval.konfidence.cloud" },
        ],
        id: "promo-dev-to-test-2",
        sequence: 2,
        source: { kind: "Stage", landscape: "development", name: "dev-us30" },
        status: "Waiting",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${VECTOR_REPOSITORY}//delivery-vector:2026.9.29-201426000Z`,
      },
      {
        gates: [
          {
            reason: "ActivatedOnSource",
            setAt: "2026-08-28T09:00:00Z",
            setBy: "konfidence/activated-on-source-stage-controller",
            state: "Satisfied",
            type: "activated-on-source-stage.konfidence.cloud",
          },
          {
            message: "approved by alex.admin@example.com",
            reason: "ApprovedByUser",
            setAt: "2026-08-28T09:15:00Z",
            setBy: "alex.admin@example.com",
            state: "Satisfied",
            type: "explicit-approval.konfidence.cloud",
          },
        ],
        id: "promo-dev-to-test-1",
        sequence: 1,
        source: { kind: "Stage", landscape: "development", name: "dev-us30" },
        status: "Succeeded",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${VECTOR_REPOSITORY}//delivery-vector:2026.9.24-151030000Z`,
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
        sequence: 1,
        source: { kind: "Stage", landscape: "test", name: "test-eu20" },
        status: "Ready",
        target: { kind: "Stage", landscape: "production", name: "prod-eu30" },
        vector: `${VECTOR_REPOSITORY}//delivery-vector:2026.9.18-094500000Z`,
      },
    ],
    source: { kind: "Stage", landscape: "test", name: "test-eu20" },
    target: { kind: "Stage", landscape: "production", name: "prod-eu30" },
  },
];

const populatedProject: ProjectFixture = {
  artifactContents,
  artifactDeployments,
  landscapes,
  project: paymentsProject,
  resolvableVectorDeployments,
  roles: ["admin", "dev"],
  stages,
  vectorContents,
  vectorDeployments,
  vectorPromotionConfigs,
};

const emptyProject = (project: Project, roles: string[]): ProjectFixture => ({
  artifactContents: [],
  artifactDeployments: [],
  landscapes: [],
  project,
  resolvableVectorDeployments: [],
  roles,
  stages: [],
  vectorContents: [],
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
        resolvableVectorDeployments: resolvableVectorDeployments.slice(0, 1),
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
