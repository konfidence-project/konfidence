import { describe, expect, it } from "vitest";

import { stageDetailsUrl } from "#lib/landscape/url.ts";

describe("stageDetailsUrl", () => {
  it("identifies the project, landscape, and stage", () => {
    const url = stageDetailsUrl({
      embedded: false,
      landscapeId: "eu",
      projectId: "payments",
      stageId: "dev-api",
    });

    expect(url).toBe("/projects/payments/landscape/eu/stages/dev-api");
  });

  it("preserves embedded mode for direct visits", () => {
    const url = stageDetailsUrl({
      embedded: true,
      landscapeId: "eu",
      projectId: "payments",
      stageId: "dev-api",
    });

    expect(url).toBe("/projects/payments/landscape/eu/stages/dev-api?embedded=1");
  });
});
