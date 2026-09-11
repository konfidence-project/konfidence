import { page } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import Avatar from "../Avatar/Avatar.svelte";
import TopBar from "./TopBar.svelte";
import TopBarTestContainer from "./__tests__/TopBarTestContainer.svelte";

describe("<TopBar>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders logo, switcher, and actions slots", async () => {
    render(TopBarTestContainer);
    expect(document.querySelector<HTMLElement>(".topbar__logo")?.textContent).toBe("Konfidence");
    expect(document.querySelector<HTMLElement>('[aria-label="Change project"]')).not.toBeNull();
    expect(document.querySelector<HTMLElement>('[aria-label="User menu"]')).not.toBeNull();
  });

  it("hides the hamburger by default", async () => {
    render(TopBarTestContainer);
    const trigger = document.querySelector<HTMLElement>('[data-testid="drawer-toggle"]');
    expect(trigger).toBeNull();
  });

  it("renders the hamburger and fires the callback on click", async () => {
    const spy = vi.fn();
    // Bypass the test container so we can pass a live callback that's actually invoked.
    render(TopBar, {
      actions: (() => "actions") as never,
      logo: (() => "logo") as never,
      onHamburger: spy,
    });
    const trigger = document.querySelector<HTMLButtonElement>('[data-testid="drawer-toggle"]');
    expect(trigger).not.toBeNull();
    trigger?.click();
    expect(spy).toHaveBeenCalledOnce();
  });

  for (const mode of ["light", "dark"] as const) {
    for (const layout of ["desktop", "mobile"] as const) {
      it(`renders ${mode} ${layout}`, async () => {
        await page.viewport(layout === "desktop" ? 720 : 400, 96);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(TopBarTestContainer, { withHamburger: layout === "mobile" });
        await expect.element(page.getByTestId("topbar-root")).toMatchScreenshot();
      });
    }
  }
});

// Placate the lint rule about unused imports — Avatar is referenced by the
// test container and hence bundled; an explicit consumer here would
// duplicate the coverage without added value.
void Avatar;
