import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import StatusBadgeHarness from "./StatusBadgeHarness.svelte";

const STATUSES = [
  "healthy",
  "warning",
  "degraded",
  "error",
  "promoting",
  "deploying",
  "queued",
] as const;

describe("<StatusBadge>", () => {
  for (const status of STATUSES) {
    it(`maps status ${status} to class .badge--${status}`, async () => {
      render(StatusBadgeHarness, { label: status, status });
      const el = page.getByText(status);
      await expect.element(el).toHaveClass("badge");
      await expect.element(el).toHaveClass(`badge--${status}`);
      await expect.element(el).toHaveAttribute("data-status", status);
    });
  }

  it("renders the visible label (auditability rule)", async () => {
    render(StatusBadgeHarness, { label: "Healthy", status: "healthy" });
    await expect.element(page.getByText("Healthy")).toBeInTheDocument();
  });

  it("hides the leading dot when showDot=false", async () => {
    render(StatusBadgeHarness, { label: "Healthy", showDot: false, status: "healthy" });
    const el = page.getByText("Healthy");
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const dots = (el.element() as HTMLElement).querySelectorAll(".dot");
    expect(dots.length).toBe(0);
  });
});
