import "../../app.css";
import { page } from "vitest/browser";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import type { ArtifactDeployment } from "#lib/konfidence-api/types.ts";
import ArtifactDeploymentsTable from "./ArtifactDeploymentsTable.svelte";
import { toArtifactDeploymentRows } from "./deployments.js";

const deployment = (id: string, version: string): ArtifactDeployment => ({
  artifact: {
    componentName: "payments-api",
    componentVersion: version,
    repository: "registry.example",
  },
  id,
  landscapeId: "development",
  stageIds: ["dev"],
  status: "ArtifactDeployed",
  vectorDeploymentIds: [],
});

const buildRows = (artifactDeployments: ArtifactDeployment[]) =>
  toArtifactDeploymentRows({
    artifactDeployments,
    landscapes: [{ id: "development", name: "Development" }],
    stages: [{ id: "dev", landscapeId: "development", name: "Development" }],
    vectorDeployments: [],
  });

const rows = buildRows([
  deployment("artifact-two", "2.0.0"),
  deployment("artifact-one", "1.0.0"),
  deployment("artifact-three", "3.0.0"),
]);

const rowIds = () =>
  Array.from(document.querySelectorAll("[data-testid='artifact-row']"), (row) =>
    row.getAttribute("data-row-id"),
  );

describe("<ArtifactDeploymentsTable>", () => {
  it("cycles sorting through ascending, descending and the original order", async () => {
    await render(ArtifactDeploymentsTable, { onSelect: vi.fn(), rows });
    const versionButton = page.getByRole("button", { name: /^Version/ });
    const versionHeader = page.getByRole("columnheader", { name: /^Version/ });

    await expect.poll(rowIds).toEqual(["artifact-two", "artifact-one", "artifact-three"]);
    await versionButton.click();
    await expect.poll(rowIds).toEqual(["artifact-one", "artifact-two", "artifact-three"]);
    await expect.element(versionHeader).toHaveAttribute("aria-sort", "ascending");

    await versionButton.click();
    await expect.poll(rowIds).toEqual(["artifact-three", "artifact-two", "artifact-one"]);
    await expect.element(versionHeader).toHaveAttribute("aria-sort", "descending");

    await versionButton.click();
    await expect.poll(rowIds).toEqual(["artifact-two", "artifact-one", "artifact-three"]);
    await expect.element(versionHeader).toHaveAttribute("aria-sort", "none");
  });

  it("updates rows from changed props without losing the active sort", async () => {
    const props = { onSelect: vi.fn(), rows };
    const rendered = await render(ArtifactDeploymentsTable, props);
    await page.getByRole("button", { name: /^Version/ }).click();
    await expect.poll(rowIds).toEqual(["artifact-one", "artifact-two", "artifact-three"]);

    await rendered.rerender({
      ...props,
      rows: [...rows.slice(0, 1), ...buildRows([deployment("artifact-new", "0.5.0")])],
    });
    await expect.poll(rowIds).toEqual(["artifact-new", "artifact-two"]);
    await expect
      .element(page.getByRole("columnheader", { name: /^Version/ }))
      .toHaveAttribute("aria-sort", "ascending");

    await rendered.rerender({ ...props, rows: [] });
    await expect.poll(rowIds).toEqual([]);
  });
});
