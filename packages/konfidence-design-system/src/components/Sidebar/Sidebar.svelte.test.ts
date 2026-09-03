import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import SidebarHarness from "./SidebarHarness.svelte";

describe("<Sidebar>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders default slot content", async () => {
    render(SidebarHarness);
    await expect.element(page.getByRole("link", { name: "Landscape" })).toBeInTheDocument();
  });

  it("renders the footer slot when provided", async () => {
    render(SidebarHarness);
    await expect.element(page.getByTestId("sidebar-footer-text")).toBeInTheDocument();
  });

  it("still renders a footer container when the snippet returns nothing", async () => {
    render(SidebarHarness, { withFooter: false });
    const footer = document.querySelector<HTMLElement>(".sidebar__footer");
    // The footer slot renders an empty container when the snippet is provided
    // but returns nothing; the container itself is still present. Assert the
    // absence of visible footer text instead.
    expect(footer?.textContent?.trim()).toBe("");
  });

  for (const mode of ["light", "dark"] as const) {
    for (const withMobileSwitcher of [false, true]) {
      it(`renders ${mode} ${withMobileSwitcher ? "with mobile switcher" : "default"}`, async () => {
        await page.viewport(260, 440);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(SidebarHarness, { withMobileSwitcher });
        await expect.element(page.getByTestId("sidebar-root")).toMatchScreenshot();
      });
    }
  }
});
