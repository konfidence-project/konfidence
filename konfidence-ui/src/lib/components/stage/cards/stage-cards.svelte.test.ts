import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import type { Stage } from "$lib/stages";
import StageCard from "./StageCard.svelte";

const stage: Stage = {
  generation: 3,
  id: "production-api",
  landscapeId: "production",
  landscapeName: "Production",
  name: "Production API",
  status: "Active",
  vector: {
    componentName: "api",
    componentVersion: "2.14.0-a3f2c9",
    repository: "ghcr.io/konfidence/mock",
  },
};

describe("StageCard", () => {
  it("renders OpenAPI stage data with status and phases", async () => {
    const screen = await render(StageCard, { props: { stage } });

    await expect.element(screen.getByText("Production API", { exact: true })).toBeVisible();
    await expect.element(screen.getByText("Live", { exact: true })).toBeVisible();
    await expect.element(screen.getByText("v2.14.0", { exact: true })).toBeVisible();
        await expect.element(screen.getByText("Deploy", { exact: true }).first()).toBeVisible();
        await expect.element(screen.getByText("Tasks", { exact: true }).first()).toBeVisible();
        await expect.element(screen.getByText("Active", { exact: true }).first()).toBeVisible();
  });

  it("shows a non-active stage as deploying", async () => {
    const screen = await render(StageCard, {
      props: { stage: { ...stage, status: "MigrationTasks" } },
    });

    await expect.element(screen.getByText("Deploying", { exact: true })).toBeVisible();
  });
});
