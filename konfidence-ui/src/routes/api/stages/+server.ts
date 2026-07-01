import { json } from "@sveltejs/kit";

import type { Stage, StageList } from "$lib/stages";

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const timestamp = (minutesAgo: number) =>
  new Date(Date.now() - minutesAgo * 60_000).toISOString();

const stage = (
  name: string,
  vector: string,
  conditions: Stage["status"]["conditions"],
  vectorHistory: string[],
): Stage => ({
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Stage",
  metadata: {
    name,
    namespace: "konfidence-system",
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
      namespace: "konfidence-system",
    },
  },
});

const stages: Stage[] = [
  stage(
    "dev",
    "github.com/konfidence-project/payment-service:1.18.3",
    [
      {
        type: "VectorDeploymentCreated",
        status: "True",
        lastTransitionTime: timestamp(42),
        reason: "DeploymentCreated",
        message: "VectorDeployment was created successfully.",
      },
      {
        type: "VectorDeployed",
        status: "True",
        lastTransitionTime: timestamp(38),
        reason: "ArtifactsDeployed",
        message: "All vector artifacts are deployed in the stage.",
      },
      {
        type: "VectorMigrated",
        status: "True",
        lastTransitionTime: timestamp(35),
        reason: "MigrationComplete",
        message: "Migration tasks completed successfully.",
      },
      {
        type: "Ready",
        status: "True",
        lastTransitionTime: timestamp(34),
        reason: "StageReady",
        message: "Stage is ready for traffic.",
      },
    ],
    [
      "github.com/konfidence-project/payment-service:1.18.1",
      "github.com/konfidence-project/payment-service:1.18.2",
    ],
  ),
  stage(
    "staging",
    "github.com/konfidence-project/checkout:2.7.0-rc.4",
    [
      {
        type: "VectorDeploymentCreated",
        status: "True",
        lastTransitionTime: timestamp(24),
        reason: "DeploymentCreated",
        message: "VectorDeployment was created successfully.",
      },
      {
        type: "VectorDeployed",
        status: "True",
        lastTransitionTime: timestamp(20),
        reason: "ArtifactsDeployed",
        message: "All vector artifacts are deployed in the stage.",
      },
      {
        type: "VectorMigrated",
        status: "Unknown",
        lastTransitionTime: timestamp(16),
        reason: "MigrationRunning",
        message: "Database migration job is still running.",
      },
      {
        type: "Ready",
        status: "Unknown",
        lastTransitionTime: timestamp(16),
        reason: "WaitingForMigration",
        message: "Stage will become ready after migrations finish.",
      },
    ],
    ["github.com/konfidence-project/checkout:2.6.9"],
  ),
  stage(
    "prod-eu",
    "github.com/konfidence-project/catalog:4.3.1",
    [
      {
        type: "FetchFailed",
        status: "False",
        lastTransitionTime: timestamp(9),
        reason: "FetchSucceeded",
        message: "Vector metadata was fetched successfully.",
      },
      {
        type: "VectorDeploymentCreated",
        status: "False",
        lastTransitionTime: timestamp(7),
        reason: "CreateFailed",
        message:
          "VectorDeployment admission failed because a rollout window is closed.",
      },
      {
        type: "Ready",
        status: "False",
        lastTransitionTime: timestamp(7),
        reason: "DeploymentBlocked",
        message: "Stage is blocked until the deployment can be created.",
      },
    ],
    [
      "github.com/konfidence-project/catalog:4.2.8",
      "github.com/konfidence-project/catalog:4.3.0",
    ],
  ),
];

export const GET = async () => {
  await wait(2000 + Math.random() * 500);

  if (Math.random() < 0.25) {
    return json(
      { message: "Mock stage API failed while fetching stages." },
      { status: 500 },
    );
  }

  return json({
    apiVersion: "star.konfidence.cloud/v1alpha1",
    kind: "StageList",
    items: stages,
  } satisfies StageList);
};
