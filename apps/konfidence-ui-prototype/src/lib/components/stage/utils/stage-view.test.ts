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
  vector: "2.14.0-a3f2c9",
});

describe("stage presentation", () => {
  it("maps the deployment status", () => {
    expect(getStageStatusLabel(createStage("DeploymentCreated"))).toEqual({
      label: "Deploying",
      tone: "deploying",
    });
  });

  it("maps the deployment status to its progress phase", () => {
    const phases = getPhases(createStage("DeploymentCreated"));
    expect(phases.map((phase) => phase.label)).toEqual(["Deploy", "Tasks", "Active"]);
    expect(phases.map((phase) => phase.state)).toEqual(["cur", "idle", "idle"]);
  });

  it("uses landscape and generation data supplied by the OpenAPI view model", () => {
    const stage = createStage("DeploymentCreated");
    expect(getLandscapeLabel(stage)).toBe("PRODUCTION");
    expect(getChips(stage)).toEqual([{ label: "generation", value: 3 }]);
  });
});

describe("splitVector", () => {
  it("returns a placeholder for empty input", () => {
    expect(splitVector("")).toEqual({ version: "—" });
  });

  it("splits a digest from a component version", () => {
    expect(splitVector("1.2.3@abcdef1234")).toEqual({
      hash: "abcdef1234",
      version: "v1.2.3",
    });
  });

  it("extracts a trailing revision", () => {
    expect(splitVector("1.2.3-deadbeef")).toEqual({
      hash: "deadbeef",
      version: "v1.2.3",
    });
  });
});
