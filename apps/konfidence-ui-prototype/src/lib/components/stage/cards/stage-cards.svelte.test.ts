import "@ui5/webcomponents-icons/dist/AllIcons.js";

import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import type { Stage } from "$lib/stages";
import StageCard from "../StageCard.svelte";

const stage: Stage = {
  generation: 3,
  id: "production-api",
  landscapeId: "production",
  landscapeName: "Production",
  name: "Production API",
  status: "DeploymentCreated",
  vector: "2.14.0-a3f2c9",
};

describe("StageCard", () => {
  for (const variant of ["custom", "fiori", "fiori-mockup"] as const) {
    it(`renders OpenAPI stage data with the ${variant} variant`, async () => {
      const screen = await render(StageCard, { props: { stage, variant } });

      await expect.element(screen.getByText("Production API", { exact: true })).toBeVisible();
      await expect.element(screen.getByText("Deploying", { exact: true })).toBeVisible();
      await expect.element(screen.getByText("v2.14.0", { exact: true })).toBeVisible();
      await expect.element(screen.getByTitle("Deploy", { exact: true }).first()).toBeVisible();
      await expect.element(screen.getByTitle("Tasks", { exact: true }).first()).toBeVisible();
      await expect.element(screen.getByTitle("Active", { exact: true }).first()).toBeVisible();
    });
  }
});
