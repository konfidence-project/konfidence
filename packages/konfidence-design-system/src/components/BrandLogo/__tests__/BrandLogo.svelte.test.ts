import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import "../../../styles/test.css";
import BrandLogoFixture from "./BrandLogoFixture.svelte";

describe("<BrandLogo>", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.classList.remove("dark");
  });

  it("exposes one named image and sizes its SVG with the forwarded class", async () => {
    await page.viewport(800, 240);
    await render(BrandLogoFixture);
    const image = page.getByRole("img", { name: "Konfidence" });
    await expect.element(image).toBeVisible();
    expect(document.querySelectorAll('[role="img"][aria-label="Konfidence"]')).toHaveLength(1);
    const span = document.querySelector<HTMLElement>('[role="img"][aria-label="Konfidence"]');
    const svg = span?.querySelector("svg");
    expect(span?.getBoundingClientRect().width).toBe(448);
    expect(svg?.getBoundingClientRect().width).toBe(448);
    expect(svg?.getBoundingClientRect().height).toBeCloseTo((448 * 1200) / 10_329, 0);
  });

  it("updates the wordmark fill when dark mode changes", async () => {
    await render(BrandLogoFixture);
    const path = document.querySelector<SVGPathElement>('[role="img"] svg path');
    expect(path).not.toBeNull();
    expect(getComputedStyle(path!).fill).toBe("rgb(0, 0, 0)");
    document.documentElement.classList.add("dark");
    expect(getComputedStyle(path!).fill).toBe("rgb(255, 255, 255)");
  });

  for (const mode of ["light", "dark"] as const) {
    it(`renders the ${mode} logo`, async () => {
      await page.viewport(800, 240);
      document.documentElement.setAttribute("data-theme", "konfidence");
      document.documentElement.classList.toggle("dark", mode === "dark");
      await render(BrandLogoFixture);
      await expect.element(page.getByTestId("logo-background")).toMatchScreenshot();
    });
  }
});
