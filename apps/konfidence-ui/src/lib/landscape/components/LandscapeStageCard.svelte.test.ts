import "../../../app.css";
import { page } from "vitest/browser";
import { expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import type { Stage } from "$lib/landscape/landscapeApi";
import LandscapeStageCard from "$lib/landscape/components/LandscapeStageCard.svelte";
import type { StageVersion } from "$lib/landscape/stageStatus";

const active: StageVersion = {
  id: "dev-api-v1",
  stageGeneration: 1,
  status: "Ready",
  vector: "registry.example.com/payments:1.0.0",
};
const stage: Stage = {
  activeStageVersion: active,
  id: "dev-api",
  landscapeId: "primary",
  name: "dev-api",
};
const props = {
  category: "Dev",
  href: "/projects/payments/landscape/primary/stages/dev-api",
  landscapeName: "Primary",
};

const scenarios: Record<string, Stage> = {
  activating: {
    ...stage,
    targetStageVersion: { ...active, id: "new", status: "ActivatingVector" },
  },
  active: { ...stage, targetStageVersion: active },
  deploying: {
    ...stage,
    targetStageVersion: { ...active, id: "new", status: "DeployingVector" },
  },
  failed: {
    ...stage,
    targetStageVersion: { ...active, id: "new", status: "Failed" },
  },
  migrating: {
    ...stage,
    targetStageVersion: { ...active, id: "new", status: "MigratingVector" },
  },
  pending: {
    ...stage,
    activeStageVersion: undefined,
    targetStageVersion: { ...active, id: "new", status: "PendingDeployment" },
  },
};

it("keeps the active version visible when a new target fails and responds to prop changes", async () => {
  const rendered = render(LandscapeStageCard, { ...props, stage: scenarios.failed });
  const link = page.getByRole("link", {
    name: "View details for stage dev-api in Primary, Dev",
  });
  await expect.element(link).toHaveAttribute("href", props.href);
  await expect.element(page.getByRole("group", { exact: true, name: "Failed" })).toBeVisible();
  await expect.element(page.getByText(active.id)).toBeVisible();
  await expect.element(page.getByText("live", { exact: true })).not.toBeInTheDocument();
  await rendered.rerender({ ...props, stage: scenarios.active });
  await expect.element(page.getByText("live", { exact: true })).toBeVisible();
  await expect.element(page.getByText("Matches target")).toBeVisible();
});

it("advances the active phase as the target status progresses", async () => {
  const rendered = render(LandscapeStageCard, { ...props, stage: scenarios.deploying });
  const card = page.getByRole("link").element();
  const activeSeg = () => card.querySelector(".stage-progress__seg--active");
  expect(activeSeg()?.textContent?.trim()).toBe("Deploy");
  await rendered.rerender({ ...props, stage: scenarios.migrating });
  expect(activeSeg()?.textContent?.trim()).toBe("Migrate");
  await rendered.rerender({ ...props, stage: scenarios.activating });
  expect(activeSeg()?.textContent?.trim()).toBe("Activate");
  await rendered.rerender({ ...props, stage: scenarios.failed });
  expect(card.querySelector(".stage-progress__seg--active")).toBeNull();
  const failed = card.querySelector(".stage-progress__seg--failed");
  expect(failed?.textContent?.trim()).toBe("Activate");
});

it("colours the top stripe from the target status", async () => {
  const rendered = render(LandscapeStageCard, { ...props, stage: scenarios.deploying });
  const link = page.getByRole("link").element() as HTMLElement;
  expect(link.getAttribute("data-status")).toBe("deploying");
  await rendered.rerender({ ...props, stage: scenarios.failed });
  expect(link.getAttribute("data-status")).toBe("error");
  await rendered.rerender({ ...props, stage: scenarios.active });
  expect(link.getAttribute("data-status")).toBe("healthy");
  await rendered.rerender({ ...props, stage: scenarios.pending });
  expect(link.getAttribute("data-status")).toBe("neutral");
});
