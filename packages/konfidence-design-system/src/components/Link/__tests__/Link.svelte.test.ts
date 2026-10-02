import { page } from "vitest/browser";
import { afterEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../../apps/konfidence-ui/src/app.css";
import LinkFixture from "./LinkFixture.svelte";

describe("<Link>", () => {
  it("renders an action as a button", async () => {
    render(LinkFixture, { label: "Clear filters" });
    const link = page.getByRole("button", { name: "Clear filters" });
    await expect.element(link).toHaveClass("link");
    await expect.element(link).toHaveAttribute("type", "button");
  });

  it("invokes the action click handler", async () => {
    const onclick = vi.fn();
    render(LinkFixture, { label: "Clear filters", onclick });
    await page.getByRole("button", { name: "Clear filters" }).click();
    expect(onclick).toHaveBeenCalledOnce();
  });

  it("does not fire onclick when disabled", async () => {
    const onclick = vi.fn();
    render(LinkFixture, { disabled: true, label: "Clear filters", onclick });
    const link = page.getByRole("button", { name: "Clear filters" });
    await expect.element(link).toBeDisabled();
    await link.click({ force: true });
    expect(onclick).not.toHaveBeenCalled();
  });

  it("renders navigation as an anchor", async () => {
    render(LinkFixture, { href: "/projects", label: "Projects" });
    await expect
      .element(page.getByRole("link", { name: "Projects" }))
      .toHaveAttribute("href", "/projects");
  });

  it("forwards aria-label and extra classes", async () => {
    render(LinkFixture, {
      "aria-label": "View projects",
      class: "custom-link",
      href: "/projects",
      label: "Projects",
    });
    const link = page.getByRole("link", { name: "View projects" });
    await expect.element(link).toHaveClass("link");
    await expect.element(link).toHaveClass("custom-link");
  });

  describe("variant snapshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      it(`matches the ${mode} screenshot`, async () => {
        await page.viewport(320, 240);
        document.documentElement.setAttribute("data-theme", "konfidence");
        document.documentElement.setAttribute("data-mode", mode);
        render(LinkFixture, { href: "/projects", label: "View projects" });
        await expect.element(page.getByRole("link", { name: "View projects" })).toMatchScreenshot();
      });
    }
  });
});
