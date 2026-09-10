import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import NavGroupTestContainer from "./__tests__/NavGroupTestContainer.svelte";

describe("<NavGroup>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders the label and children", async () => {
    render(NavGroupTestContainer);
    await expect.element(page.getByText("Delivery")).toBeInTheDocument();
    await expect.element(page.getByRole("link", { name: "Landscape" })).toBeInTheDocument();
    await expect
      .element(page.getByRole("link", { name: /Vector Deployments/ }))
      .toBeInTheDocument();
  });

  it("hides the label div when no label is provided", async () => {
    render(NavGroupTestContainer, { hideLabel: true });
    const label = document.querySelector<HTMLElement>(".nav-group__label");
    expect(label).toBeNull();
  });

  for (const mode of ["light", "dark"] as const) {
    it(`renders in ${mode} mode`, async () => {
      await page.viewport(240, 200);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      render(NavGroupTestContainer);
      await expect.element(page.getByTestId("nav-group-root")).toMatchScreenshot();
    });
  }
});
