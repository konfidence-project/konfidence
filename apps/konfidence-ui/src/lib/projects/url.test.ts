import { describe, expect, it } from "vitest";
import { stageDetailsUrl, stageOverlayUrl } from "$lib/projects/url";

describe("stageOverlayUrl", () => {
  it("points at the landscape with the stage search param", () => {
    const url = stageOverlayUrl({ embedded: false, projectId: "payments", stageId: "dev-api" });
    expect(url).toBe("/projects/payments/landscape?stage=dev-api");
  });

  it("preserves the embedded flag alongside the stage param", () => {
    const url = stageOverlayUrl({ embedded: true, projectId: "payments", stageId: "dev-api" });
    const parsed = new URL(url, "http://x");
    expect(parsed.searchParams.get("embedded")).toBe("1");
    expect(parsed.searchParams.get("stage")).toBe("dev-api");
  });
});

describe("stageDetailsUrl", () => {
  it("points at the dedicated stage detail route", () => {
    const url = stageDetailsUrl({
      embedded: false,
      landscapeId: "eu",
      projectId: "payments",
      stageId: "dev-api",
    });
    expect(url).toBe("/projects/payments/landscape/eu/stages/dev-api");
  });

  it("preserves the embedded flag", () => {
    const url = stageDetailsUrl({
      embedded: true,
      landscapeId: "eu",
      projectId: "payments",
      stageId: "dev-api",
    });
    expect(url).toBe("/projects/payments/landscape/eu/stages/dev-api?embedded=1");
  });
});
