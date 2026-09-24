import "../../../app.css";
import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import StageVersionDetails from "$lib/landscape/components/StageVersionDetails.svelte";
import type { StageVersion } from "$lib/landscape/stageStatus";

const version: StageVersion = {
  id: "dev-api-v3",
  stageGeneration: 3,
  status: "Ready",
  vector: "registry.example.com/delivery/vector:3.0.0",
};

const statuses = [
  ["ActivatingVector", "Activating vector", "deploying"],
  ["DeployingVector", "Deploying vector", "deploying"],
  ["Failed", "Failed", "error"],
  ["MigratingVector", "Migrating vector", "deploying"],
  ["PendingDeployment", "Pending deployment", "queued"],
  ["Ready", "Ready", "healthy"],
] as const satisfies readonly [StageVersion["status"], string, string][];

describe("<StageVersionDetails>", () => {
  it("renders the version metadata in an accessible section", async () => {
    render(StageVersionDetails, { label: "Target", version });

    const section = page.getByRole("region", { name: "Target version" });
    await expect.element(section).toBeVisible();
    await expect.element(section.getByText(version.id, { exact: true })).toBeVisible();
    await expect.element(section.getByText(version.vector, { exact: true })).toBeVisible();
    await expect
      .element(section.getByText(String(version.stageGeneration), { exact: true }))
      .toBeVisible();
  });

  for (const [status, label, badge] of statuses) {
    it(`renders ${status} as ${label}`, async () => {
      render(StageVersionDetails, {
        label: "Target",
        version: { ...version, status },
      });

      await expect
        .element(page.getByText(label, { exact: true }))
        .toHaveAttribute("data-status", badge);
    });
  }

  for (const label of ["Target", "Active"]) {
    it(`renders the missing ${label.toLowerCase()} version state`, async () => {
      render(StageVersionDetails, { label });

      await expect.element(page.getByText(`No ${label.toLowerCase()} version yet.`)).toBeVisible();
    });
  }
});
