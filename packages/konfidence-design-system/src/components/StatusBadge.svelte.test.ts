import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import StatusBadgeHarness from "./StatusBadgeHarness.svelte";

// Statuses shipped with a `.badge--<name>` CSS rule in
// konfidence.custom.css. StatusBadge itself accepts any string
// (the API owns the vocabulary); this list only enumerates which
// values currently have a styled representation.
const STYLED_STATUSES = [
  "healthy",
  "warning",
  "degraded",
  "error",
  "promoting",
  "deploying",
  "queued",
] as const;

describe("<StatusBadge>", () => {
  for (const status of STYLED_STATUSES) {
    it(`maps status ${status} to class .badge--${status}`, async () => {
      render(StatusBadgeHarness, { label: status, status });
      const el = page.getByText(status);
      await expect.element(el).toHaveClass("badge");
      await expect.element(el).toHaveClass(`badge--${status}`);
      await expect.element(el).toHaveAttribute("data-status", status);
    });
  }

  it("passes an unknown status through to the class list and data attribute", async () => {
    // Unknown-to-CSS statuses stay renderable — StatusBadge is a
    // passive dispatcher, not a validator.
    render(StatusBadgeHarness, { label: "Rolling out", status: "rolling-out" });
    const el = page.getByText("Rolling out");
    await expect.element(el).toHaveClass("badge");
    await expect.element(el).toHaveClass("badge--rolling-out");
    await expect.element(el).toHaveAttribute("data-status", "rolling-out");
  });

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
