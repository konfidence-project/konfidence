import { json } from "@sveltejs/kit";

import type { Stage, StageConditionType, StageList } from "$lib/stages";

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const timestamp = (minutesAgo: number) => new Date(Date.now() - minutesAgo * 60_000).toISOString();

const stage = (
  name: string,
  namespace: string,
  vector: string,
  conditions: Stage["status"]["conditions"],
  vectorHistory: string[],
): Stage => ({
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Stage",
  metadata: {
    name,
    namespace,
    creationTimestamp: timestamp(60 * (vectorHistory.length + 1)),
    generation: vectorHistory.length + 1,
  },
  spec: { vector },
  status: {
    conditions,
    vectorHistory,
    latestVectorDeploymentRef: {
      apiGroup: "star.konfidence.cloud",
      kind: "VectorDeployment",
      name: `${name}-${vector.split(":").at(-1)?.replaceAll(".", "-")}`,
      namespace,
    },
  },
});

const condition = (
  type: StageConditionType,
  status: "True" | "False" | "Unknown",
  minutesAgo: number,
  reason: string,
  message: string,
): NonNullable<Stage["status"]["conditions"]>[number] => ({
  type,
  status,
  lastTransitionTime: timestamp(minutesAgo),
  reason,
  message,
});

const live = (baseMinutesAgo: number): Stage["status"]["conditions"] => [
  condition(
    "VectorDeploymentCreated",
    "True",
    baseMinutesAgo,
    "DeploymentCreated",
    "VectorDeployment was created successfully.",
  ),
  condition(
    "VectorDeployed",
    "True",
    baseMinutesAgo - 4,
    "ArtifactsDeployed",
    "All vector artifacts are deployed in the stage.",
  ),
  condition(
    "VectorMigrated",
    "True",
    baseMinutesAgo - 7,
    "MigrationComplete",
    "Migration tasks completed successfully.",
  ),
  condition("Ready", "True", baseMinutesAgo - 8, "StageReady", "Stage is ready for traffic."),
];

const deploying = (baseMinutesAgo: number): Stage["status"]["conditions"] => [
  condition(
    "VectorDeploymentCreated",
    "True",
    baseMinutesAgo,
    "DeploymentCreated",
    "VectorDeployment was created successfully.",
  ),
  condition(
    "VectorDeployed",
    "True",
    baseMinutesAgo - 3,
    "ArtifactsDeployed",
    "All vector artifacts are deployed in the stage.",
  ),
  condition(
    "VectorMigrated",
    "Unknown",
    baseMinutesAgo - 5,
    "MigrationRunning",
    "Database migration job is still running.",
  ),
  condition(
    "Ready",
    "Unknown",
    baseMinutesAgo - 5,
    "WaitingForMigration",
    "Stage will become ready after migrations finish.",
  ),
];

const blocked = (baseMinutesAgo: number): Stage["status"]["conditions"] => [
  condition(
    "FetchFailed",
    "False",
    baseMinutesAgo,
    "FetchSucceeded",
    "Vector metadata was fetched successfully.",
  ),
  condition(
    "VectorDeploymentCreated",
    "False",
    baseMinutesAgo - 2,
    "CreateFailed",
    "VectorDeployment admission failed because a rollout window is closed.",
  ),
  condition(
    "Ready",
    "False",
    baseMinutesAgo - 2,
    "DeploymentBlocked",
    "Stage is blocked until the deployment can be created.",
  ),
];

const stages: Stage[] = [
  stage(
    "dev-us30",
    "development",
    "github.com/konfidence-project/payment-service:2.14.0-a3f2c9",
    live(72),
    [
      "github.com/konfidence-project/payment-service:2.13.8-f8e1b2",
      "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    ],
  ),
  stage(
    "dev-eu12",
    "development",
    "github.com/konfidence-project/payment-service:2.14.0-a3f2c9",
    live(66),
    ["github.com/konfidence-project/payment-service:2.13.9-d91a77"],
  ),
  stage(
    "dev-ap30",
    "development",
    "github.com/konfidence-project/payment-service:2.14.1-b6ac10",
    live(54),
    ["github.com/konfidence-project/payment-service:2.14.0-a3f2c9"],
  ),
  stage(
    "dev-canary",
    "development",
    "github.com/konfidence-project/checkout:2.8.0-rc.2",
    deploying(31),
    ["github.com/konfidence-project/checkout:2.7.4-c44e21"],
  ),
  stage(
    "dev-sandbox",
    "development",
    "github.com/konfidence-project/catalog:4.4.0-7ab921",
    live(115),
    ["github.com/konfidence-project/catalog:4.3.1-f8e1b2"],
  ),
  stage(
    "staging-us30",
    "staging",
    "github.com/konfidence-project/payment-service:2.14.0-a3f2c9",
    live(44),
    [
      "github.com/konfidence-project/payment-service:2.13.8-f8e1b2",
      "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    ],
  ),
  stage(
    "staging-eu12",
    "staging",
    "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    blocked(19),
    ["github.com/konfidence-project/payment-service:2.13.8-f8e1b2"],
  ),
  stage(
    "prod-us30",
    "production",
    "github.com/konfidence-project/payment-service:2.13.9-d91a77",
    deploying(12),
    ["github.com/konfidence-project/payment-service:2.13.8-f8e1b2"],
  ),
  stage(
    "prod-eu12",
    "production",
    "github.com/konfidence-project/payment-service:2.13.8-f8e1b2",
    live(97),
    ["github.com/konfidence-project/payment-service:2.13.7-19bb44"],
  ),
];

export const GET = async () => {
  await wait(100 + Math.random() * 500);

  return json({
    apiVersion: "star.konfidence.cloud/v1alpha1",
    kind: "StageList",
    items: stages,
  } satisfies StageList);
};
