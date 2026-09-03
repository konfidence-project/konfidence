import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import IconButtonHarness from "./IconButtonHarness.svelte";

describe("<IconButton>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders a button with the icon", async () => {
    render(IconButtonHarness);
    const btn = page.getByRole("button", { name: "Notifications" });
    await expect.element(btn).toBeInTheDocument();
    const svg = document.querySelector<SVGElement>("button.icon-btn svg[data-icon='bell']");
    expect(svg).not.toBeNull();
  });

  it("renders a badge when provided", async () => {
    render(IconButtonHarness, { badge: 3 });
    const badge = document.querySelector<HTMLElement>(".icon-btn__badge");
    expect(badge?.textContent).toBe("3");
  });

  it("clamps large badge values", async () => {
    const OVERFLOW_COUNT = 150;
    render(IconButtonHarness, { badge: OVERFLOW_COUNT });
    const badge = document.querySelector<HTMLElement>(".icon-btn__badge");
    expect(badge?.textContent).toBe("99+");
  });

  it("defaults to type=button", async () => {
    render(IconButtonHarness);
    await expect
      .element(page.getByRole("button", { name: "Notifications" }))
      .toHaveAttribute("type", "button");
  });

  for (const mode of ["light", "dark"] as const) {
    for (const withBadge of [false, true]) {
      it(`renders ${mode} ${withBadge ? "with badge" : "default"}`, async () => {
        await page.viewport(120, 96);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(IconButtonHarness, { badge: withBadge ? 3 : undefined });
        await expect
          .element(page.getByRole("button", { name: "Notifications" }))
          .toMatchScreenshot();
      });
    }
  }
});
