import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import IconButtonTestContainer from "./__tests__/IconButtonTestContainer.svelte";

describe("<IconButton>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders a button with the icon", async () => {
    render(IconButtonTestContainer);
    const btn = page.getByRole("button", { name: "Notifications" });
    await expect.element(btn).toBeInTheDocument();
    await customElements.whenDefined("ui5-icon");
    const icon = document.querySelector<HTMLElement & { name?: string }>(
      "button.icon-btn ui5-icon",
    );
    expect(icon?.name).toBe("bell");
  });

  it("renders a badge when provided", async () => {
    render(IconButtonTestContainer, { badge: 3 });
    const badge = document.querySelector<HTMLElement>(".icon-btn__badge");
    expect(badge?.textContent).toBe("3");
  });

  it("clamps large badge values", async () => {
    const OVERFLOW_COUNT = 150;
    render(IconButtonTestContainer, { badge: OVERFLOW_COUNT });
    const badge = document.querySelector<HTMLElement>(".icon-btn__badge");
    expect(badge?.textContent).toBe("99+");
  });

  it("defaults to type=button", async () => {
    render(IconButtonTestContainer);
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
        render(IconButtonTestContainer, { badge: withBadge ? 3 : undefined });
        await expect
          .element(page.getByRole("button", { name: "Notifications" }))
          .toMatchScreenshot();
      });
    }
  }
});
