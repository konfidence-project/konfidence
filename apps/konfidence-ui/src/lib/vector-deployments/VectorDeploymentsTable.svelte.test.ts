import "../../app.css";
import { page } from "vitest/browser";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import type { VectorDeployment } from "#lib/konfidence-api/types.ts";
import VectorDeploymentsTable from "./VectorDeploymentsTable.svelte";
import { toVectorDeploymentRows } from "./deployments.js";

vi.mock("$app/state", () => ({
  page: { url: new URL("http://127.0.0.1/projects/payments/vector-deployments") },
}));

const deployment = (id: string, version: string): VectorDeployment => ({
  id,
  landscapeId: "development",
  stageId: "dev",
  status: "DeploymentReady",
  vector: {
    componentName: "payments-vector",
    componentVersion: version,
    repository: "registry.example",
  },
});

const buildRows = (vectorDeployments: VectorDeployment[]) =>
  toVectorDeploymentRows({
    artifactDeployments: [],
    landscapes: [{ id: "development", name: "Development" }],
    stages: [{ id: "dev", landscapeId: "development", name: "Development" }],
    vectorDeployments,
  });

const rows = buildRows([
  deployment("vector-two", "2.0.0"),
  deployment("vector-one", "1.0.0"),
  deployment("vector-three", "3.0.0"),
]);

const rowIds = () =>
  Array.from(document.querySelectorAll("[data-testid='vectordeployment-row']"), (row) =>
    row.getAttribute("data-row-id"),
  );

describe("<VectorDeploymentsTable>", () => {
  it("cycles sorting through ascending, descending and the original order", async () => {
    await render(VectorDeploymentsTable, { onSelect: vi.fn(), projectId: "payments", rows });
    const versionButton = page.getByRole("button", { name: /^Version/ });
    const versionHeader = page.getByRole("columnheader", { name: /^Version/ });

    await expect.poll(rowIds).toEqual(["vector-two", "vector-one", "vector-three"]);
    await versionButton.click();
    await expect.poll(rowIds).toEqual(["vector-one", "vector-two", "vector-three"]);
    await expect.element(versionHeader).toHaveAttribute("aria-sort", "ascending");

    await versionButton.click();
    await expect.poll(rowIds).toEqual(["vector-three", "vector-two", "vector-one"]);
    await expect.element(versionHeader).toHaveAttribute("aria-sort", "descending");

    await versionButton.click();
    await expect.poll(rowIds).toEqual(["vector-two", "vector-one", "vector-three"]);
    await expect.element(versionHeader).toHaveAttribute("aria-sort", "none");
  });

  it("updates rows from changed props without losing the active sort", async () => {
    const props = { onSelect: vi.fn(), projectId: "payments", rows };
    const rendered = await render(VectorDeploymentsTable, props);
    await page.getByRole("button", { name: /^Version/ }).click();
    await expect.poll(rowIds).toEqual(["vector-one", "vector-two", "vector-three"]);

    await rendered.rerender({
      ...props,
      rows: [...rows.slice(0, 1), ...buildRows([deployment("vector-new", "0.5.0")])],
    });
    await expect.poll(rowIds).toEqual(["vector-new", "vector-two"]);
    await expect
      .element(page.getByRole("columnheader", { name: /^Version/ }))
      .toHaveAttribute("aria-sort", "ascending");

    await rendered.rerender({ ...props, rows: [] });
    await expect.poll(rowIds).toEqual([]);
  });
});
