import { describe, expect, it } from "vitest";
import { promotionStatuses } from "$lib/landscape/promotionStatus";

describe("promotionStatuses", () => {
  it("maps every lifecycle state to a tone and label", () => {
    expect(promotionStatuses.Succeeded).toMatchObject({ badge: "healthy", label: "Succeeded" });
    expect(promotionStatuses.Failed).toMatchObject({ badge: "error", label: "Failed" });
    expect(promotionStatuses.InProgress).toMatchObject({ badge: "deploying", label: "In progress" });
    expect(promotionStatuses.Waiting).toMatchObject({ badge: "warning", label: "Waiting" });
    expect(promotionStatuses.Ready).toMatchObject({ badge: "promoting", label: "Ready" });
    expect(promotionStatuses.Blocked).toMatchObject({ badge: "degraded", label: "Blocked" });
    expect(promotionStatuses.Superseded).toMatchObject({ badge: "queued", label: "Superseded" });
  });

  it("animates only genuinely in-flight promotions", () => {
    expect(promotionStatuses.InProgress.animated).toBe(true);
    for (const status of ["Succeeded", "Failed", "Waiting", "Ready", "Blocked", "Superseded"] as const) {
      expect(promotionStatuses[status].animated).toBe(false);
    }
  });

  it("exposes literal Tailwind classes keyed to the tone", () => {
    expect(promotionStatuses.Succeeded.stroke).toBe("stroke-[var(--status-healthy-solid)]!");
    expect(promotionStatuses.Succeeded.arrowFill).toBe("fill-[var(--status-healthy-solid)]");
    expect(promotionStatuses.Failed.pill).toContain("var(--status-error-fg)");
    expect(promotionStatuses.Failed.dot).toBe("bg-[var(--status-error-solid)]");
  });
});
