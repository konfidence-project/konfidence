import { describe, expect, it } from "vitest";
import { load } from "./+page.ts";

describe("vector deployment URL filters", () => {
  it.each([
    ["", undefined],
    ["?landscapeId=primary&deployment=detail&embedded=1", "primary"],
    ["?landscapeId=false", "false"],
    ["?landscapeId=primary&_data=unrelated", "primary"],
    ["?landscapeId=first&landscapeId=last", "last"],
  ])("loads API filters from %s", async (search, landscapeId) => {
    const event = {
      params: { projectId: "payments" },
      url: new URL(`http://localhost/projects/payments/vector-deployments${search}`),
    } as Parameters<typeof load>[0];
    expect(await load(event)).toEqual({ landscapeId, projectId: "payments" });
  });
});
