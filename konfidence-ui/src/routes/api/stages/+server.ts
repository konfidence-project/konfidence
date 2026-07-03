import type { Stage, StageConditionType, StageList } from "$lib/stages";
import { json } from "@sveltejs/kit";

const MILLISECONDS_PER_MINUTE = 60_000;
const MINUTES_PER_HOUR = 60;
const LIVE_DEPLOY_OFFSET = 4;
const LIVE_MIGRATE_OFFSET = 7;
const LIVE_READY_OFFSET = 8;
const DEPLOYING_DEPLOY_OFFSET = 3;
const DEPLOYING_MIGRATE_OFFSET = 5;
const BLOCKED_OFFSET = 2;
const MIN_RESPONSE_DELAY = 100;
const RANDOM_RESPONSE_DELAY = 500;

const stageAges = {
  devAp30: 54,
  devCanary: 31,
  devEu12: 66,
  devSandbox: 115,
  devUs30: 72,
  prodEu12: 97,
  prodUs30: 12,
  stagingEu12: 19,
  stagingUs30: 44,
};

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const timestamp = (minutesAgo: number) =>
  new Date(Date.now() - minutesAgo * MILLISECONDS_PER_MINUTE).toISOString();

interface StageFixture {
  conditions: Stage["status"]["conditions"];
  name: string;
  namespace: string;
  vector: string;
  vectorHistory: string[];
}

interface ConditionFixture {
  message: string;
  minutesAgo: number;
  reason: string;
  status: "True" | "False" | "Unknown";
  type: StageConditionType;
}

const stage = ({ conditions, name, namespace, vector, vectorHistory }: StageFixture): Stage => ({
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Stage",
  metadata: {
    creationTimestamp: timestamp(MINUTES_PER_HOUR * (vectorHistory.length + 1)),
    generation: vectorHistory.length + 1,
    name,
    namespace,
  },
  spec: { vector },
  status: {
    conditions,
    latestVectorDeploymentRef: {
      apiGroup: "star.konfidence.cloud",
      kind: "VectorDeployment",
      name: `${name}-${vector.split(":").at(-1)?.replaceAll(".", "-")}`,
      namespace,
    },
    vectorHistory,
  },
});

const condition = ({
  message,
  minutesAgo,
  reason,
  status,
  type,
}: ConditionFixture): NonNullable<Stage["status"]["conditions"]>[number] => ({
  lastTransitionTime: timestamp(minutesAgo),
  message,
  reason,
  status,
  type,
});

const live = (baseMinutesAgo: number): Stage["status"]["conditions"] => [
  condition({
    message: "VectorDeployment was created successfully.",
    minutesAgo: baseMinutesAgo,
    reason: "DeploymentCreated",
    status: "True",
    type: "VectorDeploymentCreated",
  }),
  condition({
    message: "All vector artifacts are deployed in the stage.",
    minutesAgo: baseMinutesAgo - LIVE_DEPLOY_OFFSET,
    reason: "ArtifactsDeployed",
    status: "True",
    type: "VectorDeployed",
  }),
  condition({
    message: "Migration tasks completed successfully.",
    minutesAgo: baseMinutesAgo - LIVE_MIGRATE_OFFSET,
    reason: "MigrationComplete",
    status: "True",
    type: "VectorMigrated",
  }),
  condition({
    message: "Stage is ready for traffic.",
    minutesAgo: baseMinutesAgo - LIVE_READY_OFFSET,
    reason: "StageReady",
    status: "True",
    type: "Ready",
  }),
];

const deploying = (baseMinutesAgo: number): Stage["status"]["conditions"] => [
  condition({
    message: "VectorDeployment was created successfully.",
    minutesAgo: baseMinutesAgo,
    reason: "DeploymentCreated",
    status: "True",
    type: "VectorDeploymentCreated",
  }),
  condition({
    message: "All vector artifacts are deployed in the stage.",
    minutesAgo: baseMinutesAgo - DEPLOYING_DEPLOY_OFFSET,
    reason: "ArtifactsDeployed",
    status: "True",
    type: "VectorDeployed",
  }),
  condition({
    message: "Database migration job is still running.",
    minutesAgo: baseMinutesAgo - DEPLOYING_MIGRATE_OFFSET,
    reason: "MigrationRunning",
    status: "Unknown",
    type: "VectorMigrated",
  }),
  condition({
    message: "Stage will become ready after migrations finish.",
    minutesAgo: baseMinutesAgo - DEPLOYING_MIGRATE_OFFSET,
    reason: "WaitingForMigration",
    status: "Unknown",
    type: "Ready",
  }),
];

const blocked = (baseMinutesAgo: number): Stage["status"]["conditions"] => [
  condition({
    message: "Vector metadata was fetched successfully.",
    minutesAgo: baseMinutesAgo,
    reason: "FetchSucceeded",
    status: "False",
    type: "FetchFailed",
  }),
  condition({
    message: "VectorDeployment admission failed because a rollout window is closed.",
    minutesAgo: baseMinutesAgo - BLOCKED_OFFSET,
    reason: "CreateFailed",
    status: "False",
    type: "VectorDeploymentCreated",
  }),
  condition({
    message: "Stage is blocked until the deployment can be created.",
    minutesAgo: baseMinutesAgo - BLOCKED_OFFSET,
    reason: "DeploymentBlocked",
    status: "False",
    type: "Ready",
  }),
];

const stages: Stage[] = [
  stage({
    conditions: live(stageAges.devUs30),
    name: "dev-us30",
    namespace: "development",
    vector: "github.com/konfidence-project/payment-service:2.14.0-a3f2c9",
    vectorHistory: [
      "github.com/konfidence-project/payment-service:2.13.8-f8e1b2",
      "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    ],
  }),
  stage({
    conditions: live(stageAges.devEu12),
    name: "dev-eu12",
    namespace: "development",
    vector: "github.com/konfidence-project/payment-service:2.14.0-a3f2c9",
    vectorHistory: ["github.com/konfidence-project/payment-service:2.13.9-d91a77"],
  }),
  stage({
    conditions: live(stageAges.devAp30),
    name: "dev-ap30",
    namespace: "development",
    vector: "github.com/konfidence-project/payment-service:2.14.1-b6ac10",
    vectorHistory: ["github.com/konfidence-project/payment-service:2.14.0-a3f2c9"],
  }),
  stage({
    conditions: deploying(stageAges.devCanary),
    name: "dev-canary",
    namespace: "development",
    vector: "github.com/konfidence-project/checkout:2.8.0-rc.2",
    vectorHistory: ["github.com/konfidence-project/checkout:2.7.4-c44e21"],
  }),
  stage({
    conditions: live(stageAges.devSandbox),
    name: "dev-sandbox",
    namespace: "development",
    vector: "github.com/konfidence-project/catalog:4.4.0-7ab921",
    vectorHistory: ["github.com/konfidence-project/catalog:4.3.1-f8e1b2"],
  }),
  stage({
    conditions: live(stageAges.stagingUs30),
    name: "staging-us30",
    namespace: "staging",
    vector: "github.com/konfidence-project/payment-service:2.14.0-a3f2c9",
    vectorHistory: [
      "github.com/konfidence-project/payment-service:2.13.8-f8e1b2",
      "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    ],
  }),
  stage({
    conditions: blocked(stageAges.stagingEu12),
    name: "staging-eu12",
    namespace: "staging",
    vector: "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    vectorHistory: ["github.com/konfidence-project/payment-service:2.13.8-f8e1b2"],
  }),
  stage({
    conditions: deploying(stageAges.prodUs30),
    name: "prod-us30",
    namespace: "production",
    vector: "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    vectorHistory: ["github.com/konfidence-project/payment-service:2.13.8-f8e1b2"],
  }),
  stage({
    conditions: live(stageAges.prodEu12),
    name: "prod-eu12",
    namespace: "production",
    vector: "github.com/konfidence-project/payment-service:2.13.8-f8e1b2",
    vectorHistory: ["github.com/konfidence-project/payment-service:2.13.7-19bb44"],
  }),
];

export const GET = async () => {
  await wait(MIN_RESPONSE_DELAY + Math.random() * RANDOM_RESPONSE_DELAY);

  return json({
    apiVersion: "star.konfidence.cloud/v1alpha1",
    items: stages,
    kind: "StageList",
  } satisfies StageList);
};
