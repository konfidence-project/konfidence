import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import StageCardFixture from "./StageCardFixture.svelte";
import type { StagePhaseItem } from "../../StagePhase/types.js";
import type { StageCardStatusRole } from "../types.js";
import "../../../../../../apps/konfidence-ui/src/app.css";

const VECTOR = "registry.example.com/payments:1.0.0";
const LONG_VECTOR = `registry.example.com:5000/konfidence/payments/international-reconciliation@sha256:${"abcdef0123456789".repeat(4)}`;
const ACTIVE_ID = "dev-api-v1";

interface Scenario {
  activeVersionText: string;
  live?: boolean;
  phaseAriaLabel: string;
  phases: StagePhaseItem[];
  statusRole: StageCardStatusRole;
  targetVector?: string;
  title?: string;
}

const scenarios: Record<string, Scenario> = {
  activating: {
    activeVersionText: ACTIVE_ID,
    phaseAriaLabel: "Activating vector",
    phases: [
      { label: "Deploy", state: "done" },
      { label: "Migrate", state: "done" },
      { label: "Activate", state: "active" },
    ],
    statusRole: "deploying",
    targetVector: VECTOR,
  },
  active: {
    activeVersionText: "Matches target",
    live: true,
    phaseAriaLabel: "Ready",
    phases: [
      { label: "Deploy", state: "done" },
      { label: "Migrate", state: "done" },
      { label: "Activate", state: "done" },
    ],
    statusRole: "healthy",
    targetVector: VECTOR,
  },
  "active only": {
    activeVersionText: ACTIVE_ID,
    phaseAriaLabel: "No target",
    phases: [
      { label: "Deploy", state: "pending" },
      { label: "Migrate", state: "pending" },
      { label: "Activate", state: "pending" },
    ],
    statusRole: "neutral",
  },
  deploying: {
    activeVersionText: ACTIVE_ID,
    phaseAriaLabel: "Deploying vector",
    phases: [
      { label: "Deploy", state: "active" },
      { label: "Migrate", state: "pending" },
      { label: "Activate", state: "pending" },
    ],
    statusRole: "deploying",
    targetVector: VECTOR,
  },
  empty: {
    activeVersionText: "Nothing active yet",
    phaseAriaLabel: "No target",
    phases: [
      { label: "Deploy", state: "pending" },
      { label: "Migrate", state: "pending" },
      { label: "Activate", state: "pending" },
    ],
    statusRole: "neutral",
  },
  failed: {
    activeVersionText: ACTIVE_ID,
    phaseAriaLabel: "Failed",
    phases: [
      { label: "Deploy", state: "done" },
      { label: "Migrate", state: "done" },
      { label: "Activate", state: "failed" },
    ],
    statusRole: "error",
    targetVector: VECTOR,
  },
  "long references": {
    activeVersionText: "Matches target",
    live: true,
    phaseAriaLabel: "Ready",
    phases: [
      { label: "Deploy", state: "done" },
      { label: "Migrate", state: "done" },
      { label: "Activate", state: "done" },
    ],
    statusRole: "healthy",
    targetVector: LONG_VECTOR,
    title: "dev-payments-international-reconciliation-service",
  },
  migrating: {
    activeVersionText: ACTIVE_ID,
    phaseAriaLabel: "Migrating vector",
    phases: [
      { label: "Deploy", state: "done" },
      { label: "Migrate", state: "active" },
      { label: "Activate", state: "pending" },
    ],
    statusRole: "deploying",
    targetVector: VECTOR,
  },
  pending: {
    activeVersionText: "Nothing active yet",
    phaseAriaLabel: "Pending deployment",
    phases: [
      { label: "Deploy", state: "pending" },
      { label: "Migrate", state: "pending" },
      { label: "Activate", state: "pending" },
    ],
    statusRole: "neutral",
    targetVector: VECTOR,
  },
  "ready but not active": {
    activeVersionText: ACTIVE_ID,
    phaseAriaLabel: "Ready",
    phases: [
      { label: "Deploy", state: "done" },
      { label: "Migrate", state: "done" },
      { label: "Activate", state: "done" },
    ],
    statusRole: "neutral",
    targetVector: VECTOR,
  },
};

const DEFAULT_PROPS = {
  ariaLabel: "View details for stage dev-api in Primary, Dev",
  href: "/projects/payments/landscape/primary/stages/dev-api",
  title: "dev-api",
};

const propsFor = (scenario: Scenario) => ({
  ...DEFAULT_PROPS,
  ...scenario,
  title: scenario.title ?? DEFAULT_PROPS.title,
});

describe("<StageCard>", () => {
  it("renders as a labelled anchor pointing at href", async () => {
    render(StageCardFixture, propsFor(scenarios.deploying));
    const link = page.getByRole("link", {
      name: "View details for stage dev-api in Primary, Dev",
    });
    await expect.element(link).toHaveAttribute("href", DEFAULT_PROPS.href);
  });

  it("exposes the status role via the data-status attribute", async () => {
    const rendered = render(StageCardFixture, propsFor(scenarios.deploying));
    const link = page.getByRole("link").element() as HTMLElement;
    expect(link.getAttribute("data-status")).toBe("deploying");
    await rendered.rerender(propsFor(scenarios.failed));
    expect(link.getAttribute("data-status")).toBe("error");
    await rendered.rerender(propsFor(scenarios.active));
    expect(link.getAttribute("data-status")).toBe("healthy");
    await rendered.rerender(propsFor(scenarios["active only"]));
    expect(link.getAttribute("data-status")).toBe("neutral");
  });

  it("renders the live pill only when `live` is true", async () => {
    const rendered = render(StageCardFixture, propsFor(scenarios.active));
    await expect.element(page.getByText("live", { exact: true })).toBeVisible();
    await rendered.rerender({ ...propsFor(scenarios.deploying), live: false });
    await expect.element(page.getByText("live", { exact: true })).not.toBeInTheDocument();
  });

  it("falls back to a placeholder when targetVector is missing", async () => {
    render(StageCardFixture, propsFor(scenarios["active only"]));
    await expect.element(page.getByText("No target version yet")).toBeVisible();
  });

  it("labels the phase strip via `phaseAriaLabel`", async () => {
    render(StageCardFixture, propsFor(scenarios.failed));
    await expect.element(page.getByRole("group", { exact: true, name: "Failed" })).toBeVisible();
  });

  describe("variant screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      for (const [name, scenario] of Object.entries(scenarios)) {
        it(`renders ${mode} ${name}`, async () => {
          await page.viewport(304, 300);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(StageCardFixture, propsFor(scenario));
          await expect.element(page.getByRole("link")).toMatchScreenshot();
        });
      }

      it(`renders ${mode} selected`, async () => {
        await page.viewport(304, 300);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(StageCardFixture, { ...propsFor(scenarios.deploying), selected: true });
        await expect.element(page.getByRole("link")).toMatchScreenshot();
      });
    }
  });
});
