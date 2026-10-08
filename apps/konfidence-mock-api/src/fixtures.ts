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

// Maps a stage version's lifecycle status onto the matching vector deployment status.
const vectorStatusForStage = (status: StageVersion["status"]): VectorDeployment["status"] => {
  if (status === "Ready") {
    return "DeploymentReady";
  }
  if (status === "Failed") {
    return "DeploymentFailed";
  }
  return "DeployingVector";
};

// Extracts the "<component>:<version>" tail from a vector reference
// (`<repo>//<component>:<version>`) so deployments can echo the stage version.
const vectorVersion = (vector: string): string => vector.split(":").at(-1) ?? "0.0.0";

const paymentsProject = {
  id: "payments-platform",
  name: "Payments Platform",
} satisfies Project;
const identityProject = {
  id: "identity-service",
  name: "Identity Service",
} satisfies Project;

// Vector + artifact deployments are derived from the stages below so the three datasets stay in sync.

const landscapes: Landscape[] = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
  { id: "production", name: "Production" },
  // Intentionally stage-less to exercise the empty-landscape placeholder path.
  { id: "staging", name: "Staging" },
];

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

// A compact landscape (three dev, two test, two prod stages) exercising every promotion state.
const stages: Stage[] = [
  stage("dev-us30", "dev-us30", "development", {
    activeStageVersion: stageVersion("dev-us30", 5, "Ready", "2026.8.5"),
    targetStageVersion: stageVersion("dev-us30", 5, "Ready", "2026.8.5"),
  }),
  stage("dev-eu10", "dev-eu10", "development", {
    activeStageVersion: stageVersion("dev-eu10", 5, "Ready", "2026.8.5"),
    targetStageVersion: stageVersion("dev-eu10", 6, "DeployingVector", "2026.8.6"),
  }),
  stage("dev-rollback", "dev-rollback", "development", {
    activeStageVersion: stageVersion("dev-rollback", 1, "Ready", "2026.8.1"),
    targetStageVersion: stageVersion("dev-rollback", 2, "Failed", "2026.8.7"),
  }),
  stage("test-eu20", "test-eu20", "test", {
    activeStageVersion: stageVersion("test-eu20", 4, "Ready", "2026.8.4"),
    targetStageVersion: stageVersion("test-eu20", 4, "Ready", "2026.8.4"),
  }),
  stage("test-us10", "test-us10", "test", {
    activeStageVersion: stageVersion("test-us10", 4, "Ready", "2026.8.4"),
    targetStageVersion: stageVersion("test-us10", 5, "MigratingVector", "2026.8.5"),
  }),
  stage("prod-eu30", "prod-eu30", "production", {
    activeStageVersion: stageVersion("prod-eu30", 3, "Ready", "2026.8.3"),
    targetStageVersion: stageVersion("prod-eu30", 3, "Ready", "2026.8.3"),
  }),
  stage("prod-us40", "prod-us40", "production", {
    targetStageVersion: stageVersion("prod-us40", 1, "PendingDeployment", "2026.8.8"),
  }),
];

const component = (componentName: string, componentVersion: string) => ({
  componentName,
  componentVersion,
  repository: REPOSITORY,
});

// The stage version actually being deployed: the in-flight target if present, else the active one.
const deployedVersion = (item: Stage): StageVersion | undefined =>
  item.targetStageVersion ?? item.activeStageVersion;

// One vector deployment per stage with a version, so every graphed stage has a matching deployment.
const vectorDeployments: VectorDeployment[] = stages.flatMap((item) => {
  const version = deployedVersion(item);
  if (version === undefined) {
    return [];
  }
  return [
    {
      id: `vector-${item.id}-1`,
      landscapeId: item.landscapeId,
      stageId: item.id,
      status: vectorStatusForStage(version.status),
      vector: {
        componentName: "delivery-vector",
        componentVersion: vectorVersion(version.vector),
        repository: REPOSITORY,
      },
    },
  ];
});

// A small catalogue of app artifact component names, cycled across stages.
const ARTIFACT_NAMES = ["payments-api", "payments-ui", "ledger-worker"] as const;

const artifactStatusForStage = (
  status: StageVersion["status"],
): ArtifactDeployment["status"] => {
  if (status === "Ready") {
    return "AppHealthy";
  }
  if (status === "Failed") {
    return "ArtifactFetched";
  }
  return "ArtifactDeployed";
};

// Two artifact deployments per vector deployment (an app plus a supporting component).
// The component version mirrors the deployed vector version for distinct, sortable rows.
const artifactDeployments: ArtifactDeployment[] = vectorDeployments.flatMap(
  (deployment, index) => {
    const stageStatus =
      stages.find(({ id }) => id === deployment.stageId)?.targetStageVersion?.status ??
      stages.find(({ id }) => id === deployment.stageId)?.activeStageVersion?.status ??
      "Ready";
    const version = deployment.vector.componentVersion;
    const primary = component(ARTIFACT_NAMES[index % ARTIFACT_NAMES.length]!, version);
    const secondary = component(ARTIFACT_NAMES[(index + 1) % ARTIFACT_NAMES.length]!, version);
    return [
      {
        artifact: primary,
        id: `artifact-${deployment.stageId}-1`,
        landscapeId: deployment.landscapeId,
        stageIds: [deployment.stageId],
        status: artifactStatusForStage(stageStatus),
        vectorDeploymentIds: [deployment.id],
      },
      {
        artifact: secondary,
        id: `artifact-${deployment.stageId}-2`,
        landscapeId: deployment.landscapeId,
        stageIds: [deployment.stageId],
        status: "ArtifactFetched",
        vectorDeploymentIds: [deployment.id],
      },
    ];
  },
);

const vectorPromotionConfigs: VectorPromotionConfig[] = [
  {
    id: "delivery-vector-dev-to-test",
    keepLastPromotions: 5,
    // Newest-first (descending sequence) to mirror the real API.
    // Newest wins: Succeeded (seq 3) beats the older runs, so dev->test renders green.
    promotions: [
      {
        approval: {
          approvedAt: "2026-08-28T09:15:00Z",
          approvedBy: "alex.admin@example.com",
        },
        id: "promo-dev-to-test-3",
        requireApproval: true,
        sequence: 3,
        source: { kind: "Stage", landscape: "development", name: "dev-us30" },
        status: "Succeeded",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.4`,
      },
      {
        id: "promo-dev-to-test-2",
        requireApproval: true,
        sequence: 2,
        source: { kind: "Stage", landscape: "development", name: "dev-us30" },
        status: "Superseded",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.3`,
      },
    ],
    source: { kind: "Stage", landscape: "development", name: "dev-us30" },
    target: { kind: "Stage", landscape: "test", name: "test-eu20" },
    ttlAfterFinished: "168h0m0s",
  },
  {
    // Ready but cannot execute (target unresolved), so test->prod renders as Blocked (orange).
    id: "delivery-vector-test-to-prod",
    keepLastPromotions: 3,
    promotions: [
      {
        id: "promo-test-to-prod-1",
        requireApproval: false,
        sequence: 1,
        source: { kind: "Stage", landscape: "test", name: "test-eu20" },
        status: "Blocked",
        target: { kind: "Stage", landscape: "production", name: "prod-eu30" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.3`,
      },
    ],
    source: { kind: "Stage", landscape: "test", name: "test-eu20" },
    target: { kind: "Stage", landscape: "production", name: "prod-eu30" },
  },
  {
    // Awaiting manual approval: the dev->test edge for eu10 renders as Waiting
    // (amber).
    id: "delivery-vector-dev-eu10-to-test",
    keepLastPromotions: 3,
    promotions: [
      {
        id: "promo-dev-eu10-1",
        requireApproval: true,
        sequence: 1,
        source: { kind: "Stage", landscape: "development", name: "dev-eu10" },
        status: "Waiting",
        target: { kind: "Stage", landscape: "test", name: "test-us10" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.5`,
      },
    ],
    source: { kind: "Stage", landscape: "development", name: "dev-eu10" },
    target: { kind: "Stage", landscape: "test", name: "test-us10" },
  },
  {
    // Approved and queued for execution: the test->prod edge renders as Ready
    // (purple).
    id: "delivery-vector-test-us10-to-prod",
    keepLastPromotions: 3,
    promotions: [
      {
        id: "promo-test-us10-1",
        requireApproval: false,
        sequence: 1,
        source: { kind: "Stage", landscape: "test", name: "test-us10" },
        status: "Ready",
        target: { kind: "Stage", landscape: "production", name: "prod-us40" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.4`,
      },
    ],
    source: { kind: "Stage", landscape: "test", name: "test-us10" },
    target: { kind: "Stage", landscape: "production", name: "prod-us40" },
  },
  {
    // Last run failed: the dev->test edge for rollback renders red.
    id: "delivery-vector-dev-rollback-to-test",
    keepLastPromotions: 3,
    promotions: [
      {
        id: "promo-dev-rollback-1",
        requireApproval: false,
        sequence: 1,
        source: { kind: "Stage", landscape: "development", name: "dev-rollback" },
        status: "Failed",
        target: { kind: "Stage", landscape: "test", name: "test-eu20" },
        vector: `${REPOSITORY}//delivery-vector:2026.8.0`,
      },
    ],
    source: { kind: "Stage", landscape: "development", name: "dev-rollback" },
    target: { kind: "Stage", landscape: "test", name: "test-eu20" },
  },
  {
    // Configured but never run: the test->prod edge renders faint and dashed
    // (idle).
    id: "delivery-vector-test-us10-to-prod-eu30",
    keepLastPromotions: 3,
    promotions: [],
    source: { kind: "Stage", landscape: "test", name: "test-us10" },
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
