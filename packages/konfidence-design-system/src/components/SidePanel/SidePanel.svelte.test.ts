import { page } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import SidePanelFixture from "./__tests__/SidePanelFixture.svelte";
import "../../../../../apps/konfidence-ui/src/app.css";

describe("<SidePanel>", () => {
  it("stays closed until asked to open", async () => {
    render(SidePanelFixture);
    const panel = page.getByTestId("side-panel");
    await expect.element(panel).not.toBeVisible();
  });

  it("opens on demand and exposes an accessible title", async () => {
    render(SidePanelFixture, { open: true, title: "Artifact detail" });
    const panel = page.getByRole("dialog", { name: "Artifact detail" });
    await expect.element(panel).toBeVisible();
    await expect.element(page.getByTestId("panel-body")).toBeVisible();
  });

  it("closes when the close button is clicked", async () => {
    render(SidePanelFixture, { open: true });
    await page.getByTestId("side-panel-close").click();
    await expect.element(page.getByTestId("side-panel")).not.toBeVisible();
  });

  it("renders the footer snippet when provided", async () => {
    render(SidePanelFixture, { open: true });
    await expect.element(page.getByTestId("panel-footer")).toBeInTheDocument();
  });

  it("invokes onClose when the user closes the panel", async () => {
    const onClose = vi.fn();
    render(SidePanelFixture, { onClose, open: true });
    await page.getByTestId("side-panel-close").click();
    await expect.element(page.getByTestId("side-panel")).not.toBeVisible();
    expect(onClose).toHaveBeenCalled();
  });

  describe("mode screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      it(`renders the open drawer in ${mode} mode`, async () => {
        await page.viewport(560, 480);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(SidePanelFixture, {
          open: true,
          title: "payments-api@3.4.1",
        });
        await expect
          .element(page.getByRole("dialog", { name: "payments-api@3.4.1" }))
          .toMatchScreenshot();
      });
    }
  });
});
