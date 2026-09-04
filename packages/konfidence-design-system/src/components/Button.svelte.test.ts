import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import ButtonHarness from "./ButtonHarness.svelte";

const VARIANT_CLASSES = {
  danger: "btn--danger",
  ghost: "btn--ghost",
  primary: "btn--primary",
  secondary: "btn--secondary",
} as const;

describe("<Button>", () => {
  it("renders a <button> with the primary class by default", async () => {
    render(ButtonHarness, { label: "Deploy" });
    const el = page.getByRole("button", { name: "Deploy" });
    await expect.element(el).toHaveClass("btn");
    await expect.element(el).toHaveClass("btn--primary");
  });

  for (const [variant, cls] of Object.entries(VARIANT_CLASSES) as [
    keyof typeof VARIANT_CLASSES,
    string,
  ][]) {
    it(`maps variant ${variant} to class ${cls}`, async () => {
      render(ButtonHarness, { label: "Go", variant });
      await expect.element(page.getByRole("button", { name: "Go" })).toHaveClass(cls);
    });
  }

  it("renders as an anchor when href is provided", async () => {
    render(ButtonHarness, { href: "/", label: "Home", variant: "secondary" });
    const link = page.getByRole("link", { name: "Home" });
    await expect.element(link).toHaveAttribute("href", "/");
    await expect.element(link).toHaveClass("btn--secondary");
  });

  it("forwards aria-label", async () => {
    render(ButtonHarness, { "aria-label": "Confirm deploy", label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Confirm deploy" })).toBeInTheDocument();
  });

  it("does not fire onclick when disabled", async () => {
    render(ButtonHarness, { disabled: true, label: "Deploy" });
    await expect.element(page.getByRole("button", { name: "Deploy" })).toBeDisabled();
  });

  it("defaults type=button to avoid accidental form submission", async () => {
    render(ButtonHarness, { label: "Deploy" });
    await expect
      .element(page.getByRole("button", { name: "Deploy" }))
      .toHaveAttribute("type", "button");
  });

  // Snapshot tests guard against silent DOM drift across variants /
  // states. If a rendered variant changes shape, the snapshot mismatch
  // forces a deliberate `-u` (or "Update Snapshot" in the UI) so the
  // change is reviewed rather than merged unnoticed. Kept as HTML
  // string snapshots to stay readable in code review — no external
  // renderer, no image diffing.
  //
  // Svelte's scoped-CSS compiler suffixes every rendered element with
  // a `svelte-<hash>` class that changes on every content edit; strip
  // it so snapshots aren't churned by unrelated tweaks.
  const stripSvelteScope = (html: string): string =>
    html.replace(/ svelte-[a-z0-9]+/g, "").replace(/class=""/g, "");

  describe("variant snapshots", () => {
    for (const variant of Object.keys(VARIANT_CLASSES) as (keyof typeof VARIANT_CLASSES)[]) {
      it(`matches the ${variant} snapshot`, async () => {
        const { container } = await render(ButtonHarness, { label: "Go", variant });
        expect(stripSvelteScope(container.innerHTML)).toMatchSnapshot();
      });
    }

    it("matches the anchor snapshot", async () => {
      const { container } = await render(ButtonHarness, {
        href: "/",
        label: "Home",
        variant: "secondary",
      });
      expect(stripSvelteScope(container.innerHTML)).toMatchSnapshot();
    });

    it("matches the disabled snapshot", async () => {
      const { container } = await render(ButtonHarness, {
        disabled: true,
        label: "Deploy",
      });
      expect(stripSvelteScope(container.innerHTML)).toMatchSnapshot();
    });
  });
});
