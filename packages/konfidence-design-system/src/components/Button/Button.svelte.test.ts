import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../apps/konfidence-ui/src/app.css";
import ButtonTestContainer from "./__tests__/ButtonTestContainer.svelte";

const VARIANT_CLASSES = {
  danger: "btn--danger",
  ghost: "btn--ghost",
  primary: "btn--primary",
  secondary: "btn--secondary",
} as const;

describe("<Button>", () => {
  it("renders a <button> with the primary class by default", async () => {
    render(ButtonTestContainer, { label: "Deploy" });
    const el = page.getByRole("button", { name: "Deploy" });
    await expect.element(el).toHaveClass("btn");
    await expect.element(el).toHaveClass("btn--primary");
  });

  for (const [variant, cls] of Object.entries(VARIANT_CLASSES) as [
    keyof typeof VARIANT_CLASSES,
    string,
  ][]) {
    it(`maps variant ${variant} to class ${cls}`, async () => {
      render(ButtonTestContainer, { label: "Go", variant });
      await expect.element(page.getByRole("button", { name: "Go" })).toHaveClass(cls);
    });
  }

  it("renders as an anchor when href is provided", async () => {
    render(ButtonTestContainer, { href: "/", label: "Home", variant: "secondary" });
    const link = page.getByRole("link", { name: "Home" });
    await expect.element(link).toHaveAttribute("href", "/");
    await expect.element(link).toHaveClass("btn--secondary");
  });

  it("forwards aria-label", async () => {
    render(ButtonTestContainer, { "aria-label": "Confirm deploy", label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Confirm deploy" })).toBeInTheDocument();
  });

  it("does not fire onclick when disabled", async () => {
    render(ButtonTestContainer, { disabled: true, label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Deploy" })).toBeDisabled();
  });

  it("defaults type=button to avoid accidental form submission", async () => {
    render(ButtonTestContainer, { label: "Deploy" });
    await expect
      .element(page.getByRole("button", { name: "Deploy" }))
      .toHaveAttribute("type", "button");
  });

  describe("variant snapshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      for (const variant of Object.keys(VARIANT_CLASSES) as (keyof typeof VARIANT_CLASSES)[]) {
        it(`matches the ${mode} ${variant} screenshot`, async () => {
          await page.viewport(320, 240);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(ButtonTestContainer, { label: "Deploy", variant });
          await expect.element(page.getByRole("button", { name: "Deploy" })).toMatchScreenshot();
        });
      }
    }
  });
});
