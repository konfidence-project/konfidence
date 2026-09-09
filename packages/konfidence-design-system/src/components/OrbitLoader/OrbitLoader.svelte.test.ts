import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import OrbitLoader from "./OrbitLoader.svelte";
import "../../../../../apps/konfidence-ui/src/app.css";

describe("<OrbitLoader>", () => {
  it("exposes a live-region with the default label", async () => {
    render(OrbitLoader);
    const el = page.getByRole("status");
    await expect.element(el).toHaveAttribute("aria-live", "polite");
    await expect.element(el).toHaveTextContent("Loading");
  });

  it("respects a custom label", async () => {
    render(OrbitLoader, { label: "Signing out\u2026" });
    await expect.element(page.getByRole("status")).toHaveTextContent("Signing out\u2026");
  });

  describe("label screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    for (const mode of ["light", "dark"]) {
      for (const label of [undefined, "Signing out\u2026"]) {
        it(`renders ${mode} ${label === undefined ? "default" : "custom"} label`, async () => {
          await page.viewport(320, 240);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(OrbitLoader, { label });
          await expect.element(page.getByRole("status")).toMatchScreenshot();
        });
      }
    }
  });
});
