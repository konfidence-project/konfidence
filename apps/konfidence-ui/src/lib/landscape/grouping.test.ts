import { describe, expect, it } from "vitest";
import { groupStages } from "$lib/landscape/grouping";

describe("stage categories", () => {
  it("matches prefixes case-insensitively and retains API order within each category", () => {
    const stages = ["Test-api", "DEV-z", "sandbox", "dev-a", "prod-web", "development", "dev"].map(
      (name) => ({ id: name, landscapeId: "primary", name }),
    );
    const groups = groupStages(stages);
    expect(
      Object.fromEntries(
        Object.entries(groups).map(([category, items]) => [
          category,
          items.map(({ name }) => name),
        ]),
      ),
    ).toEqual({
      dev: ["DEV-z", "dev-a"],
      other: ["sandbox", "development", "dev"],
      prod: ["prod-web"],
      test: ["Test-api"],
    });
  });

  it("retains all four empty categories", () => {
    expect(groupStages([])).toEqual({ dev: [], other: [], prod: [], test: [] });
  });
});
