import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import OrbitLoader from "../components/OrbitLoader.svelte";

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
});
