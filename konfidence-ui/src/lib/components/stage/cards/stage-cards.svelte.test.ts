import "@ui5/webcomponents-icons/dist/AllIcons.js";

import { describe, expect, it } from "vitest";

import type { Stage } from "$lib/stages.js";
import StageCardCustom from "./StageCardCustom.svelte";
import StageCardFiori from "./StageCardFiori.svelte";
import StageCardFioriHybrid from "./StageCardFioriHybrid.svelte";
import { render } from "vitest-browser-svelte";

const healthyStage: Stage = {
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Stage",
  metadata: {
    creationTimestamp: "2024-01-01T00:00:00Z",
    generation: 3,
    name: "dev-us30",
    namespace: "development",
  },
  spec: { vector: "github.com/konfidence/payment-service:2.14.0@a3f2c9" },
  status: {
    conditions: [
      {
        lastTransitionTime: "2024-01-01T00:00:00Z",
        message: "VectorDeployment was created successfully.",
        reason: "DeploymentCreated",
        status: "True",
        type: "VectorDeploymentCreated",
      },
      {
        lastTransitionTime: "2024-01-01T00:01:00Z",
        message: "All vector artifacts are deployed.",
        reason: "ArtifactsDeployed",
        status: "True",
        type: "VectorDeployed",
      },
      {
        lastTransitionTime: "2024-01-01T00:02:00Z",
        message: "Migration completed.",
        reason: "MigrationComplete",
        status: "True",
        type: "VectorMigrated",
      },
      {
        lastTransitionTime: "2024-01-01T00:03:00Z",
        message: "Stage is ready for traffic.",
        reason: "StageReady",
        status: "True",
        type: "Ready",
      },
    ],
    latestVectorDeploymentRef: {
      apiGroup: "star.konfidence.cloud",
      kind: "VectorDeployment",
      name: "dev-us30-2-14-0-a3f2c9",
      namespace: "development",
    },
    vectorHistory: [
      "github.com/konfidence/payment-service:2.13.8@f8e1b2",
      "github.com/konfidence/payment-service:2.13.9@d91a77",
    ],
  },
};

describe("StageCardCustom", () => {
  it("renders stage name, status, and vector for a healthy stage", async () => {
    const screen = await render(StageCardCustom, {
      props: { selected: false, stage: healthyStage },
    });

    await expect.element(screen.getByText("dev-us30", { exact: true })).toBeVisible();
    await expect.element(screen.getByText("Live")).toBeVisible();
    await expect.element(screen.getByText("v2.14.0")).toBeVisible();
  });
});

describe("StageCardFiori", () => {
  it("renders stage name, status, and vector for a healthy stage", async () => {
    const screen = await render(StageCardFiori, {
      props: { stage: healthyStage },
    });

    await expect.element(screen.getByText("dev-us30", { exact: true })).toBeVisible();
    await expect.element(screen.getByText("Live")).toBeVisible();
    await expect.element(screen.getByText("v2.14.0")).toBeVisible();
  });
});

describe("StageCardFioriHybrid", () => {
  it("renders stage name, status, and vector for a healthy stage", async () => {
    const screen = await render(StageCardFioriHybrid, {
      props: { selected: false, stage: healthyStage },
    });

    await expect.element(screen.getByText("dev-us30", { exact: true })).toBeVisible();
    await expect.element(screen.getByText("Live")).toBeVisible();
    await expect.element(screen.getByText("v2.14.0")).toBeVisible();
  });
});
