import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import NavItemTestContainer from "./__tests__/NavItemTestContainer.svelte";

describe("<NavItem>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders an anchor with the href", async () => {
    render(NavItemTestContainer, { href: "/foo", label: "Landscape" });
    await expect
      .element(page.getByRole("link", { name: "Landscape" }))
      .toHaveAttribute("href", "/foo");
  });

  it("applies .nav-item--active and aria-current when active", async () => {
    render(NavItemTestContainer, { active: true, label: "Landscape" });
    const link = document.querySelector<HTMLAnchorElement>("a.nav-item");
    expect(link?.classList.contains("nav-item--active")).toBe(true);
    expect(link?.getAttribute("aria-current")).toBe("page");
  });

  it("renders a leading icon when set", async () => {
    render(NavItemTestContainer, { icon: "landscape", label: "Landscape" });
    const svg = document.querySelector<SVGElement>("a.nav-item svg[data-icon='landscape']");
    expect(svg).not.toBeNull();
  });

  it("renders a trailing badge when set", async () => {
    render(NavItemTestContainer, { badge: 3, label: "Promotions" });
    const badge = document.querySelector<HTMLElement>("a.nav-item .nav-item__badge");
    expect(badge?.textContent).toBe("3");
  });

  it("clamps large badge values", async () => {
    const OVERFLOW_COUNT = 150;
    render(NavItemTestContainer, { badge: OVERFLOW_COUNT, label: "Feed" });
    const badge = document.querySelector<HTMLElement>("a.nav-item .nav-item__badge");
    expect(badge?.textContent).toBe("99+");
  });

  describe("variant screenshots", () => {
    for (const mode of ["light", "dark"] as const) {
      for (const variant of ["default", "active", "with-icon", "with-badge"] as const) {
        it(`renders ${mode} ${variant}`, async () => {
          await page.viewport(280, 60);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          const props = {
            active: variant === "active",
            badge: variant === "with-badge" ? 3 : undefined,
            icon: variant === "with-icon" ? ("landscape" as const) : undefined,
            label: "Landscape",
          };
          render(NavItemTestContainer, props);
          await expect.element(page.getByRole("link", { name: /Landscape/ })).toMatchScreenshot();
        });
      }
    }
  });
});
