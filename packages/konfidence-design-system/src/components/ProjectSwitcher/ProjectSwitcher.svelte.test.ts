import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import ProjectSwitcherTestContainer from "./__tests__/ProjectSwitcherTestContainer.svelte";

describe("<ProjectSwitcher>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders the name inside .project-switch__name", async () => {
    render(ProjectSwitcherTestContainer, { name: "Aurora" });
    const trigger = page.getByTestId("project-switch");
    await expect.element(trigger).toBeInTheDocument();
    const inner = document.querySelector<HTMLElement>(".project-switch__name");
    expect(inner?.textContent).toBe("Aurora");
  });

  it("defaults to type=button", async () => {
    render(ProjectSwitcherTestContainer);
    await expect.element(page.getByTestId("project-switch")).toHaveAttribute("type", "button");
  });

  for (const mode of ["light", "dark"] as const) {
    it(`renders in ${mode} mode`, async () => {
      await page.viewport(220, 96);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      render(ProjectSwitcherTestContainer, { name: "Konfidence" });
      await expect.element(page.getByTestId("project-switch")).toMatchScreenshot();
    });
  }
});
