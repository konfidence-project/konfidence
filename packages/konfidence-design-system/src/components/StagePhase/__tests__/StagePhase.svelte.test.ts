import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import StagePhaseFixture from "./StagePhaseFixture.svelte";
import type { StagePhaseItem } from "../types.js";
import "../../../../../../apps/konfidence-ui/src/app.css";

const DEFAULT_PHASES: StagePhaseItem[] = [
  { label: "Deploy", state: "done" },
  { label: "Migrate", state: "active" },
  { label: "Activate", state: "pending" },
];

describe("<StagePhase>", () => {
  it("renders one segment per phase, in order", () => {
    render(StagePhaseFixture, { phases: DEFAULT_PHASES });
    const segments = document.querySelectorAll(".stage-progress__seg");
    expect(segments.length).toBe(3);
    expect(segments[0].getAttribute("data-state")).toBe("done");
    expect(segments[1].getAttribute("data-state")).toBe("active");
    expect(segments[2].getAttribute("data-state")).toBe("pending");
  });

  it("applies state-specific modifier classes", () => {
    render(StagePhaseFixture, {
      phases: [
        { label: "one", state: "done" },
        { label: "two", state: "active" },
        { label: "three", state: "failed" },
        { label: "four", state: "pending" },
      ],
    });
    const [first, second, third, fourth] = document.querySelectorAll(".stage-progress__seg");
    expect(first.classList.contains("stage-progress__seg--done")).toBe(true);
    expect(second.classList.contains("stage-progress__seg--active")).toBe(true);
    expect(third.classList.contains("stage-progress__seg--failed")).toBe(true);
    expect(fourth.classList.contains("stage-progress__seg--pending")).toBe(true);
  });

  it("marks the active segment with aria-current=step", () => {
    render(StagePhaseFixture, { phases: DEFAULT_PHASES });
    const active = document.querySelector('[aria-current="step"]');
    expect(active).not.toBeNull();
    expect(active?.classList.contains("stage-progress__seg--active")).toBe(true);
  });

  it("hides labels in compact size", () => {
    render(StagePhaseFixture, { phases: DEFAULT_PHASES, size: "compact" });
    expect(document.querySelector(".stage-progress--compact")).not.toBeNull();
    expect(document.querySelectorAll(".stage-progress__label").length).toBe(0);
  });

  it("shows labels in default and large sizes", () => {
    render(StagePhaseFixture, { phases: DEFAULT_PHASES, size: "lg" });
    expect(document.querySelector(".stage-progress--lg")).not.toBeNull();
    expect(document.querySelectorAll(".stage-progress__label").length).toBe(3);
  });

  it("exposes the strip as a labelled group when ariaLabel is provided", async () => {
    render(StagePhaseFixture, {
      ariaLabel: "Deploying vector",
      phases: DEFAULT_PHASES,
    });
    await expect.element(page.getByRole("group", { name: "Deploying vector" })).toBeVisible();
  });

  describe("variant screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    const scenarios: Record<string, StagePhaseItem[]> = {
      "activate active": [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "done" },
        { label: "Activate", state: "active" },
      ],
      "all done": [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "done" },
        { label: "Activate", state: "done" },
      ],
      "all pending": [
        { label: "Deploy", state: "pending" },
        { label: "Migrate", state: "pending" },
        { label: "Activate", state: "pending" },
      ],
      "deploy active": [
        { label: "Deploy", state: "active" },
        { label: "Migrate", state: "pending" },
        { label: "Activate", state: "pending" },
      ],
      failed: [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "done" },
        { label: "Activate", state: "failed" },
      ],
      "migrate active": [
        { label: "Deploy", state: "done" },
        { label: "Migrate", state: "active" },
        { label: "Activate", state: "pending" },
      ],
    };

    for (const mode of ["light", "dark"]) {
      for (const size of ["compact", "default", "lg"] as const) {
        for (const [name, phases] of Object.entries(scenarios)) {
          it(`renders ${mode} ${size} ${name}`, async () => {
            await page.viewport(360, 80);
            document.documentElement.setAttribute("data-theme", "konfidence");
            document.documentElement.setAttribute("data-mode", mode);
            render(StagePhaseFixture, { phases, size });
            await expect.element(page.getByTestId("stage-phase-root")).toMatchScreenshot();
          });
        }
      }
    }
  });
});
