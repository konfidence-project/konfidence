import "../../../app.css";
import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import type { Stage } from "$lib/landscape/landscapeApi";
import StageCard from "$lib/landscape/components/StageCard.svelte";
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
  "active only": stage,
  deploying: {
    ...stage,
    targetStageVersion: { ...active, id: "new", status: "DeployingVector" },
  },
  empty: { ...stage, activeStageVersion: undefined },
  failed: {
    ...stage,
    targetStageVersion: { ...active, id: "new", status: "Failed" },
  },
  "long references": {
    ...stage,
    name: "dev-payments-international-reconciliation-service",
    targetStageVersion: {
      ...active,
      vector: `registry.example.com:5000/konfidence/payments/international-reconciliation@sha256:${"abcdef0123456789".repeat(4)}`,
    },
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
  "ready but not active": {
    ...stage,
    targetStageVersion: { ...active, id: "new" },
  },
};

afterEach(() => {
  document.documentElement.removeAttribute("data-theme");
  document.documentElement.removeAttribute("data-mode");
});

for (const mode of ["light", "dark"]) {
  describe(`stage card ${mode}`, () => {
    for (const [name, example] of Object.entries(scenarios)) {
      it(`renders ${name}`, async () => {
        await page.viewport(304, 300);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(StageCard, { ...props, stage: example });
        await expect.element(page.getByRole("link")).toMatchScreenshot();
      });
    }
  });
}

it("keeps the active version visible when a new target fails and responds to prop changes", async () => {
  const rendered = render(StageCard, { ...props, stage: scenarios.failed });
  const link = page.getByRole("link", {
    name: "View details for stage dev-api in Primary, Dev",
  });
  await expect.element(link).toHaveAttribute("href", props.href);
  await expect.element(page.getByRole("group", { exact: true, name: "Failed" })).toBeVisible();
  await expect.element(page.getByText(active.id)).toBeVisible();
  await expect.element(page.getByText("Active", { exact: true })).not.toBeInTheDocument();
  await rendered.rerender({ ...props, stage: scenarios.active });
  await expect.element(page.getByText("Active", { exact: true })).toBeVisible();
  await expect.element(page.getByText("Matches target")).toBeVisible();
});

it("moves the status dot to the running step and clears it when the failed step is unknown", async () => {
  const rendered = render(StageCard, { ...props, stage: scenarios.deploying });
  const card = page.getByRole("link").element();
  expect(card.querySelector('[aria-current="step"]')?.textContent?.trim()).toBe("Deploy");
  expect(card.querySelector('[aria-current="step"] .dot')).not.toBeNull();
  await rendered.rerender({ ...props, stage: scenarios.migrating });
  expect(card.querySelector('[aria-current="step"]')?.textContent?.trim()).toBe("Migrate");
  await rendered.rerender({ ...props, stage: scenarios.failed });
  expect(card.querySelector('[aria-current="step"]')).toBeNull();
  expect(card.querySelector("footer .dot")).toBeNull();
});
