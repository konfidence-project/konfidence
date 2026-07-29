import { describe, expect, it } from "vitest";
import type { Stage, StageStatus } from "$lib/stages";
import {
  getChips,
  getLandscapeLabel,
  getPhases,
  getStageStatusLabel,
  splitVector,
} from "./stage-view";

const createStage = (status: StageStatus): Stage => ({
  generation: 3,
  id: "production-api",
  landscapeId: "production",
  landscapeName: "Production",
  name: "Production API",
  status,
  vector: {
    componentName: "api",
    componentVersion: "2.14.0-a3f2c9",
    repository: "ghcr.io/konfidence/mock",
  },
});

describe("stage presentation", () => {
  it.each([
    ["DeploymentCreated", "Deploying", "deploying"],
    ["MigrationTasks", "Deploying", "deploying"],
    ["Active", "Live", "healthy"],
  ] as const)("maps %s to %s", (status, label, tone) => {
    expect(getStageStatusLabel(createStage(status))).toEqual({ label, tone });
  });

  it.each([
    ["DeploymentCreated", ["cur", "idle", "idle"]],
    ["MigrationTasks", ["done", "cur", "idle"]],
    ["Active", ["done", "done", "done"]],
  ] as const)("maps %s to its three progress phases", (status, states) => {
    const phases = getPhases(createStage(status));
    expect(phases.map((phase) => phase.label)).toEqual(["Deploy", "Tasks", "Active"]);
    expect(phases.map((phase) => phase.state)).toEqual(states);
  });

  it("uses landscape and generation data supplied by the OpenAPI view model", () => {
    const stage = createStage("Active");
    expect(getLandscapeLabel(stage)).toBe("PRODUCTION");
    expect(getChips(stage)).toEqual([{ label: "generation", value: 3 }]);
  });
});

describe("splitVector", () => {
  it("returns a placeholder for empty input", () => {
    expect(splitVector({ componentName: "api", componentVersion: "", repository: "repo" })).toEqual(
      { version: "—" },
    );
  });

  it("splits a digest from a component version", () => {
    expect(
      splitVector({
        componentName: "vector",
        componentVersion: "1.2.3@abcdef1234",
        repository: "registry.example.com",
      }),
    ).toEqual({
      hash: "abcdef1234",
      version: "v1.2.3",
    });
  });

  it("extracts a trailing revision", () => {
    expect(
      splitVector({
        componentName: "vector",
        componentVersion: "1.2.3-deadbeef",
        repository: "registry.example.com",
      }),
    ).toEqual({
      hash: "deadbeef",
      version: "v1.2.3",
    });
  });
});
