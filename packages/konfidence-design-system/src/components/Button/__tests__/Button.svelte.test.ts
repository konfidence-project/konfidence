import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../../../../apps/konfidence-ui/src/app.css";
import ButtonFixture from "./ButtonFixture.svelte";

const VARIANT_CLASSES = {
  danger: "btn--danger",
  ghost: "btn--ghost",
  primary: "btn--primary",
  secondary: "btn--secondary",
} as const;

describe("<Button>", () => {
  it("renders a <button> with the primary class by default", async () => {
    render(ButtonFixture, { label: "Deploy" });
    const el = page.getByRole("button", { name: "Deploy" });
    await expect.element(el).toHaveClass("btn");
    await expect.element(el).toHaveClass("btn--primary");
  });

  for (const [variant, cls] of Object.entries(VARIANT_CLASSES) as [
    keyof typeof VARIANT_CLASSES,
    string,
  ][]) {
    it(`maps variant ${variant} to class ${cls}`, async () => {
      render(ButtonFixture, { label: "Go", variant });
      await expect.element(page.getByRole("button", { name: "Go" })).toHaveClass(cls);
    });
  }

  it("renders as an anchor when href is provided", async () => {
    render(ButtonFixture, { href: "/", label: "Home", variant: "secondary" });
    const link = page.getByRole("link", { name: "Home" });
    await expect.element(link).toHaveAttribute("href", "/");
    await expect.element(link).toHaveClass("btn--secondary");
  });

  it("forwards aria-label", async () => {
    render(ButtonFixture, { "aria-label": "Confirm deploy", label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Confirm deploy" })).toBeInTheDocument();
  });

  it("does not fire onclick when disabled", async () => {
    render(ButtonFixture, { disabled: true, label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Deploy" })).toBeDisabled();
  });

  it("defaults type=button to avoid accidental form submission", async () => {
    render(ButtonFixture, { label: "Deploy" });
    await expect
      .element(page.getByRole("button", { name: "Deploy" }))
      .toHaveAttribute("type", "button");
  });

  it("renders a ui5-icon when the icon prop is set", async () => {
    render(ButtonFixture, { icon: "add", label: "Add" });
    await expect.element(page.getByRole("button", { name: "Add" })).toBeInTheDocument();
    await customElements.whenDefined("ui5-icon");
    const icon = document.querySelector<HTMLElement & { name?: string }>("button.btn ui5-icon");
    expect(icon?.getAttribute("name")).toBe("add");
  });

  it("renders no icon element when icon prop is omitted", async () => {
    render(ButtonFixture, { label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Deploy" })).toBeInTheDocument();
    const icon = document.querySelector("button.btn ui5-icon");
    expect(icon).toBeNull();
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
          render(ButtonFixture, { label: "Deploy", variant });
          await expect.element(page.getByRole("button", { name: "Deploy" })).toMatchScreenshot();
        });
      }
    }
  });
});
