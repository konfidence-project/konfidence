import { describe, expect, it } from "vitest";
import { load } from "./+page.ts";

describe("artifact deployment URL filters", () => {
  it.each([
    ["", undefined, undefined],
    [
      "?landscapeId=primary&vectorDeploymentId=vector-one&deployment=detail&embedded=1",
      "primary",
      "vector-one",
    ],
    ["?landscapeId=false&vectorDeploymentId=001", "false", "001"],
    ["?landscapeId=primary&vectorDeploymentId=vector-one&_data=unrelated", "primary", "vector-one"],
    ["?landscapeId=first&landscapeId=last", "last", undefined],
  ])("loads API filters from %s", async (search, landscapeId, vectorDeploymentId) => {
    const event = {
      params: { projectId: "payments" },
      url: new URL(`http://localhost/projects/payments/artifact-deployments${search}`),
    } as Parameters<typeof load>[0];
    expect(await load(event)).toEqual({ landscapeId, projectId: "payments", vectorDeploymentId });
  });
});
