import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import StageCardPlaceholderFixture from "./StageCardPlaceholderFixture.svelte";
import "../../../../../../apps/konfidence-ui/src/app.css";

const DEFAULT_PROPS = {
  ariaLabel: "Landscape Staging has no stages yet",
  message: "No stages yet",
  title: "Staging",
};

describe("<StageCardPlaceholder>", () => {
  it("renders the title and message", async () => {
    await render(StageCardPlaceholderFixture, DEFAULT_PROPS);
    await expect.element(page.getByText("Staging", { exact: true })).toBeVisible();
    await expect.element(page.getByText("No stages yet", { exact: true })).toBeVisible();
  });

  it("exposes the aria-label on the card region", async () => {
    await render(StageCardPlaceholderFixture, DEFAULT_PROPS);
    const card = page.getByTestId("stage-card-placeholder");
    await expect.element(card).toHaveAttribute("aria-label", DEFAULT_PROPS.ariaLabel);
  });

  it("renders an optional landscape eyebrow above the title", async () => {
    await render(StageCardPlaceholderFixture, { ...DEFAULT_PROPS, landscapeName: "Primary" });
    await expect.element(page.getByText("Primary", { exact: true })).toBeVisible();
  });

  describe("variant screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      it(`renders ${mode} default`, async () => {
        await page.viewport(304, 300);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        await render(StageCardPlaceholderFixture, DEFAULT_PROPS);
        await expect.element(page.getByTestId("stage-card-placeholder")).toMatchScreenshot();
      });

      it(`renders ${mode} with eyebrow`, async () => {
        await page.viewport(304, 300);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        await render(StageCardPlaceholderFixture, { ...DEFAULT_PROPS, landscapeName: "Primary" });
        await expect.element(page.getByTestId("stage-card-placeholder")).toMatchScreenshot();
      });
    }
  });
});
