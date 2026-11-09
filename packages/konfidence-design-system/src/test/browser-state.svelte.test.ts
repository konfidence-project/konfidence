import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../styles/test.css";
import NavItemFixture from "../components/NavItem/__tests__/NavItemFixture.svelte";

describe("browser pointer isolation", () => {
  it("allows a test to explicitly hover a component", async () => {
    await render(NavItemFixture, { label: "Landscape" });
    const link = page.getByRole("link", { name: "Landscape" });
    await link.hover();
    expect(link.element().matches(":hover")).toBe(true);
  });

  it("does not inherit the previous test's hover when taking a screenshot", async () => {
    await render(NavItemFixture, { label: "Landscape" });
    // Capturing forces Chromium to update hit testing after the DOM was replaced.
    await page.screenshot({ save: false });
    expect(page.getByRole("link", { name: "Landscape" }).element().matches(":hover")).toBe(false);
  });
});
