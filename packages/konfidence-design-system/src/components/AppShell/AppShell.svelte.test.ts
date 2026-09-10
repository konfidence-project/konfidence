import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import AppShellTestContainer from "./__tests__/AppShellTestContainer.svelte";

describe("<AppShell>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.removeAttribute("data-mode");
  });

  it("renders the topbar, sidebar, and main slot content", async () => {
    render(AppShellTestContainer);
    // Brandbar is rendered via the DS component (Tailwind classes rather than
    // the raw .brandbar class), so we assert on structure instead.
    expect(document.querySelector<HTMLElement>(".app-shell")).not.toBeNull();
    expect(document.querySelector<HTMLElement>(".topbar")).not.toBeNull();
    expect(document.querySelector<HTMLElement>(".sidebar")).not.toBeNull();
    expect(document.querySelector<HTMLElement>(".app-shell__main")).not.toBeNull();
  });

  it("renders a skip link pointing at the main content", async () => {
    render(AppShellTestContainer);
    const link = document.querySelector<HTMLAnchorElement>("a.skip-link");
    expect(link?.getAttribute("href")).toBe("#app-shell-main");
  });

  it("defaults the drawer to closed", async () => {
    render(AppShellTestContainer);
    const sidebar = document.querySelector<HTMLElement>(".app-shell__sidebar");
    expect(sidebar?.getAttribute("data-open")).toBe("false");
  });

  it("opens the drawer when the topbar hamburger is clicked", async () => {
    // Mobile viewport ensures the hamburger is visible.
    await page.viewport(400, 640);
    render(AppShellTestContainer);
    const trigger = document.querySelector<HTMLButtonElement>('[data-testid="drawer-toggle"]');
    trigger?.click();
    // Wait a microtask so Svelte flushes the reactivity update.
    await new Promise((resolve) => setTimeout(resolve, 0));
    const sidebar = document.querySelector<HTMLElement>(".app-shell__sidebar");
    expect(sidebar?.getAttribute("data-open")).toBe("true");
  });

  it("closes the drawer when the scrim is clicked", async () => {
    await page.viewport(400, 640);
    render(AppShellTestContainer);
    const trigger = document.querySelector<HTMLButtonElement>('[data-testid="drawer-toggle"]');
    trigger?.click();
    await new Promise((resolve) => setTimeout(resolve, 0));

    const scrim = document.querySelector<HTMLButtonElement>(".app-shell__scrim");
    scrim?.click();
    await new Promise((resolve) => setTimeout(resolve, 0));

    const sidebar = document.querySelector<HTMLElement>(".app-shell__sidebar");
    expect(sidebar?.getAttribute("data-open")).toBe("false");
  });

  for (const mode of ["light", "dark"] as const) {
    it(`renders ${mode} desktop`, async () => {
      await page.viewport(1024, 640);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      render(AppShellTestContainer);
      await expect.element(page.getByTestId("shell-root")).toMatchScreenshot();
    });

    it(`renders ${mode} mobile with the drawer opened`, async () => {
      await page.viewport(400, 640);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.setAttribute("data-mode", mode);
      render(AppShellTestContainer);
      const trigger = document.querySelector<HTMLButtonElement>('[data-testid="drawer-toggle"]');
      trigger?.click();
      await new Promise((resolve) => setTimeout(resolve, 250));
      await expect.element(page.getByTestId("shell-root")).toMatchScreenshot();
    });
  }
});
