import "../../../app.css";
import { page } from "vitest/browser";
import { describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import type { Landscape, Stage } from "$lib/landscape/landscapeApi";
import StageDetails from "$lib/landscape/components/StageDetails.svelte";

const landscape: Landscape = { id: "primary", name: "Primary" };
const stage: Stage = {
  activeStageVersion: {
    id: "dev-api-v1",
    stageGeneration: 1,
    status: "Ready",
    vector: "registry.example.com/delivery/vector:1.0.0",
  },
  id: "dev-api",
  landscapeId: landscape.id,
  name: "dev-api",
  targetStageVersion: {
    id: "dev-api-v2",
    stageGeneration: 2,
    status: "DeployingVector",
    vector: "registry.example.com/delivery/vector:2.0.0",
  },
};
const baseProps = {
  backHref: "/projects/payments/landscape",
  onRetry: vi.fn(),
};

describe("<StageDetails>", () => {
  it("renders the loading state", async () => {
    render(StageDetails, { ...baseProps, status: "loading" });

    await expect.element(page.getByRole("status")).toHaveTextContent("Loading stage details");
  });

  it("renders a retryable API error", async () => {
    const onRetry = vi.fn();
    render(StageDetails, {
      ...baseProps,
      error: "API request failed with status 503",
      errorStatus: 503,
      onRetry,
      status: "error",
    });

    const alert = page.getByRole("alert");
    await expect.element(alert).toHaveTextContent("Stage details could not be loaded");
    await expect.element(alert).toHaveTextContent("API request failed with status 503");
    await page.getByRole("button", { name: "Retry" }).click();
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("renders a 404 as not found without a retry action", async () => {
    render(StageDetails, {
      ...baseProps,
      errorStatus: 404,
      landscape,
      stage,
      status: "error",
    });

    await expect.element(page.getByRole("heading", { name: "Stage not found" })).toBeVisible();
    await expect
      .element(page.getByRole("link", { name: "Back to landscapes" }))
      .toHaveAttribute("href", baseProps.backHref);
    await expect.element(page.getByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("renders missing stage data as not found", async () => {
    render(StageDetails, {
      ...baseProps,
      landscape,
      status: "ready",
    });

    await expect.element(page.getByRole("heading", { name: "Stage not found" })).toBeVisible();
  });

  it("renders breadcrumbs and both stage versions when ready", async () => {
    render(StageDetails, {
      ...baseProps,
      landscape,
      stage,
      status: "ready",
    });

    await expect.element(page.getByRole("heading", { level: 1, name: stage.name })).toBeVisible();
    const breadcrumbs = page.getByRole("navigation", { name: "Breadcrumb" });
    await expect
      .element(breadcrumbs.getByRole("link", { name: "Landscapes" }))
      .toHaveAttribute("href", baseProps.backHref);
    await expect.element(breadcrumbs).toHaveTextContent(landscape.name);
    await expect.element(breadcrumbs).toHaveTextContent(stage.name);
    await expect.element(page.getByRole("region", { name: "Target version" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Active version" })).toBeVisible();
    await expect.element(page.getByText("Deploying vector", { exact: true })).toBeVisible();
    await expect.element(page.getByText("Ready", { exact: true })).toBeVisible();
  });
});
