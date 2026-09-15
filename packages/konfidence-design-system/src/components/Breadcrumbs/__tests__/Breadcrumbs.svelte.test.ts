import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import Breadcrumbs from "../Breadcrumbs.svelte";
import BreadcrumbsFixture from "./BreadcrumbsFixture.svelte";
import "../../../../../../apps/konfidence-ui/src/app.css";

describe("<Breadcrumbs>", () => {
  it("renders nothing when items is empty", () => {
    render(Breadcrumbs, { items: [] });
    expect(document.querySelector('[data-testid="breadcrumbs"]')).toBeNull();
  });

  it("renders each item as a link except the last one", () => {
    render(Breadcrumbs, {
      items: [
        { href: "/root", label: "Landscapes" },
        { href: "/root/primary", label: "Primary" },
        { label: "dev-api" },
      ],
    });
    const nav = document.querySelector('[data-testid="breadcrumbs"]');
    const links = nav?.querySelectorAll("a") ?? [];
    expect(links.length).toBe(2);
    expect(links[0].getAttribute("href")).toBe("/root");
    expect(links[1].getAttribute("href")).toBe("/root/primary");
    const current = nav?.querySelector(".crumbs__current");
    expect(current?.textContent?.trim()).toBe("dev-api");
    expect(current?.getAttribute("aria-current")).toBe("page");
  });

  it("renders intermediate items without href as plain spans", () => {
    render(Breadcrumbs, {
      items: [{ href: "/root", label: "Landscapes" }, { label: "Primary" }, { label: "dev-api" }],
    });
    const nav = document.querySelector('[data-testid="breadcrumbs"]');
    const links = nav?.querySelectorAll("a") ?? [];
    expect(links.length).toBe(1);
    expect(links[0].textContent?.trim()).toBe("Landscapes");
  });

  it("inserts a separator between crumbs", () => {
    render(Breadcrumbs, {
      items: [
        { href: "/root", label: "Landscapes" },
        { href: "/root/primary", label: "Primary" },
        { label: "dev-api" },
      ],
    });
    const seps = document.querySelectorAll('[data-testid="breadcrumbs"] .crumbs__sep');
    expect(seps.length).toBe(2);
    for (const sep of seps) {
      expect(sep.getAttribute("aria-hidden")).toBe("true");
    }
  });

  it("exposes the trail as a Breadcrumb landmark", async () => {
    render(Breadcrumbs, { items: [{ label: "Root" }] });
    await expect.element(page.getByRole("navigation", { name: "Breadcrumb" })).toBeVisible();
  });

  describe("variant screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    const scenarios = {
      "single current": [{ label: "Landscapes" }],
      "three crumbs": [
        { href: "/root", label: "Landscapes" },
        { href: "/root/primary", label: "Primary" },
        { label: "dev-api" },
      ],
      "two crumbs": [{ href: "/root", label: "Landscapes" }, { label: "Primary" }],
    } satisfies Record<string, { href?: string; label: string }[]>;

    for (const mode of ["light", "dark"]) {
      for (const [name, items] of Object.entries(scenarios)) {
        it(`renders ${mode} ${name}`, async () => {
          await page.viewport(480, 80);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(BreadcrumbsFixture, { items });
          await expect.element(page.getByTestId("breadcrumbs-root")).toMatchScreenshot();
        });
      }
    }
  });
});
