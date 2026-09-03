import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import MenuHarness from "./MenuHarness.svelte";

describe("<Menu>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders items with the .menu__item class", async () => {
    render(MenuHarness, { variant: "plain" });
    const items = document.querySelectorAll(".menu__item");
    expect(items.length).toBeGreaterThan(0);
  });

  it("marks the active item with .menu__item--active", async () => {
    render(MenuHarness, { variant: "plain" });
    const active = document.querySelector<HTMLElement>(".menu__item--active");
    expect(active?.textContent?.trim()).toBe("Konfidence");
  });

  it("marks danger items with .menu__item--danger", async () => {
    render(MenuHarness, { variant: "plain" });
    const danger = document.querySelector<HTMLElement>(".menu__item--danger");
    expect(danger?.textContent?.trim()).toBe("Delete project");
  });

  it("adds .menu--header when header=true", async () => {
    render(MenuHarness, { variant: "with-header" });
    const header = document.querySelector<HTMLElement>(".menu--header");
    expect(header).not.toBeNull();
    const name = document.querySelector<HTMLElement>(".menu__header-name");
    expect(name?.textContent?.trim()).toBe("Alex Admin");
  });

  describe("variant screenshots", () => {
    for (const mode of ["light", "dark"] as const) {
      for (const variant of ["plain", "with-header", "size-sm", "size-lg"] as const) {
        it(`renders ${mode} ${variant}`, async () => {
          await page.viewport(320, 400);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(MenuHarness, { variant });
          await expect.element(page.getByTestId("menu-root")).toMatchScreenshot();
        });
      }
    }
  });
});
