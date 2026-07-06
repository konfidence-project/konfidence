import type { Stage, StageCondition } from "$lib/stages.js";
import { describe, expect, it } from "vitest";

import {
  getChips,
  getLandscapeLabel,
  getPhases,
  getStageHealth,
  getStageStatusLabel,
  splitVector,
} from "./stage-view.js";

const PHASE_COUNT = 4;
const DEPLOYMENT_TRUNCATE_LENGTH = 20;

const condition = (
  type: StageCondition["type"],
  status: StageCondition["status"],
  overrides: Partial<StageCondition> = {},
): StageCondition => ({
  lastTransitionTime: "2024-01-01T00:00:00Z",
  message: `${type} ${status}`,
  reason: "TestReason",
  status,
  type,
  ...overrides,
});

const stage = (overrides: Partial<Stage> = {}): Stage => ({
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Stage",
  metadata: {
    creationTimestamp: "2024-01-01T00:00:00Z",
    generation: 1,
    name: "test-stage",
    namespace: "dev",
  },
  spec: { vector: "vector:1.0.0" },
  status: {},
  ...overrides,
});

describe("getStageHealth", () => {
  it("returns error when FetchFailed is True", () => {
    const currentStage = stage({
      status: {
        conditions: [condition("FetchFailed", "True")],
      },
    });
    expect(getStageHealth(currentStage)).toBe("error");
    expect(getStageStatusLabel(currentStage)).toEqual({ label: "Failed", tone: "error" });
  });

  it("returns error when Ready is False", () => {
    const currentStage = stage({
      status: {
        conditions: [condition("Ready", "False")],
      },
    });
    expect(getStageHealth(currentStage)).toBe("error");
    expect(getStageStatusLabel(currentStage)).toEqual({ label: "Failed", tone: "error" });
  });

  it("returns healthy when Ready is True", () => {
    const currentStage = stage({
      status: {
        conditions: [condition("Ready", "True")],
      },
    });
    expect(getStageHealth(currentStage)).toBe("healthy");
    expect(getStageStatusLabel(currentStage)).toEqual({ label: "Live", tone: "healthy" });
  });

  it("returns deploying when an in-flight condition is True or Unknown", () => {
    const currentStage = stage({
      status: {
        conditions: [condition("VectorDeploymentCreated", "Unknown")],
      },
    });
    expect(getStageHealth(currentStage)).toBe("deploying");
    expect(getStageStatusLabel(currentStage)).toEqual({ label: "Deploying", tone: "deploying" });
  });

  it("returns warning when no conditions indicate progress", () => {
    const currentStage = stage({ status: { conditions: [] } });
    expect(getStageHealth(currentStage)).toBe("warning");
    expect(getStageStatusLabel(currentStage)).toEqual({ label: "Pending", tone: "warning" });
  });
});

describe("getPhases", () => {
  it("returns four phases with idle states when no conditions exist", () => {
    const phases = getPhases(stage({ status: {} }));
    expect(phases).toHaveLength(PHASE_COUNT);
    expect(phases.map((phase) => phase.state)).toEqual(["idle", "idle", "idle", "idle"]);
    expect(phases.map((phase) => phase.label)).toEqual(["Deploy", "Tasks", "Migrate", "Active"]);
  });

  it("maps each condition status to the corresponding phase state", () => {
    const phases = getPhases(
      stage({
        status: {
          conditions: [
            condition("VectorDeploymentCreated", "Unknown"),
            condition("VectorDeployed", "False"),
            condition("VectorMigrated", "True"),
          ],
        },
      }),
    );
    expect(phases.map((phase) => phase.state)).toEqual(["cur", "err", "done", "idle"]);
  });

  it("surfaces message and reason from Done conditions", () => {
    const phases = getPhases(
      stage({
        status: {
          conditions: [
            condition("VectorDeploymentCreated", "True", {
              message: "deployed",
              reason: "Created",
            }),
            condition("VectorDeployed", "True", {
              message: "tasks done",
              reason: "TasksComplete",
            }),
            condition("VectorMigrated", "True", {
              message: "migrated",
              reason: "Migrated",
            }),
            condition("Ready", "True", {
              message: "ready",
              reason: "Ready",
            }),
          ],
        },
      }),
    );
    expect(phases).toEqual([
      {
        id: "VectorDeploymentCreated",
        label: "Deploy",
        message: "deployed",
        reason: "Created",
        state: "done",
      },
      {
        id: "Tasks",
        label: "Tasks",
        message: "tasks done",
        reason: "TasksComplete",
        state: "done",
      },
      {
        id: "VectorMigrated",
        label: "Migrate",
        message: "migrated",
        reason: "Migrated",
        state: "done",
      },
      {
        id: "Ready",
        label: "Active",
        message: "ready",
        reason: "Ready",
        state: "done",
      },
    ]);
  });

  it("marks the deploy phase as err and uses fetchFailed message when FetchFailed is True", () => {
    const phases = getPhases(
      stage({
        status: {
          conditions: [
            condition("VectorDeploymentCreated", "True", {
              message: "deployed",
            }),
            condition("FetchFailed", "True", {
              message: "could not fetch",
              reason: "NotFound",
            }),
          ],
        },
      }),
    );
    expect(phases[0]).toEqual({
      id: "VectorDeploymentCreated",
      label: "Deploy",
      message: "could not fetch",
      reason: "NotFound",
      state: "err",
    });
  });

  it("does not override the deploy phase when FetchFailed is not True", () => {
    const phases = getPhases(
      stage({
        status: {
          conditions: [
            condition("VectorDeploymentCreated", "True", {
              message: "deployed",
            }),
            condition("FetchFailed", "False"),
          ],
        },
      }),
    );
    expect(phases[0]).toEqual({
      id: "VectorDeploymentCreated",
      label: "Deploy",
      message: "deployed",
      reason: "TestReason",
      state: "done",
    });
  });
});

describe("splitVector", () => {
  it("returns an em dash for empty input", () => {
    expect(splitVector("")).toEqual({ version: "—" });
    expect(splitVector("   ")).toEqual({ version: "—" });
  });

  it("strips a registry/tag prefix and splits name@hash", () => {
    expect(splitVector("registry.example.com/vector:1.2.3@abcdef1234")).toEqual({
      hash: "abcdef1234",
      version: "v1.2.3",
    });
  });

  it("extracts a trailing -hash suffix when no @ separator is present", () => {
    expect(splitVector("1.2.3-deadbeef")).toEqual({
      hash: "deadbeef",
      version: "v1.2.3",
    });
  });

  it("prefixes a numeric version with v", () => {
    expect(splitVector("1.0.0")).toEqual({ version: "v1.0.0" });
  });

  it("leaves a non-numeric version untouched", () => {
    expect(splitVector("main@abc123")).toEqual({
      hash: "abc123",
      version: "main",
    });
  });
});

describe("getChips", () => {
  it("shows a versions chip and a deployment chip", () => {
    const chips = getChips(
      stage({
        status: {
          latestVectorDeploymentRef: {
            kind: "Deployment",
            name: "vector-abc123",
          },
          vectorHistory: ["v1.0.0", "v1.1.0"],
        },
      }),
    );
    expect(chips).toEqual([
      { label: "versions", value: 3 },
      { label: "deployment", value: "vector-abc123" },
    ]);
  });

  it("uses singular version label when there is exactly one version", () => {
    const chips = getChips(stage({ status: {} }));
    expect(chips).toEqual([{ label: "version", value: 1 }]);
  });

  it("uses plural version label for two versions", () => {
    const chips = getChips(stage({ status: { vectorHistory: ["v1.0.0"] } }));
    expect(chips).toEqual([{ label: "versions", value: 2 }]);
  });

  it("truncates long deployment names", () => {
    const longName = "vector-0123456789abcdef0123456789abcdef";
    const chips = getChips(
      stage({
        status: {
          latestVectorDeploymentRef: { kind: "Deployment", name: longName },
        },
      }),
    );
    expect(chips[1]?.value).toBe(`${longName.slice(0, DEPLOYMENT_TRUNCATE_LENGTH)}…`);
  });

  it("adds a fetch failed chip when FetchFailed is True", () => {
    const chips = getChips(stage({ status: { conditions: [condition("FetchFailed", "True")] } }));
    expect(chips.at(-1)).toEqual({
      label: "fetch failed",
      tone: "alert",
      value: "!",
    });
  });
});

describe("getLandscapeLabel", () => {
  it("uppercases the stage namespace", () => {
    const { metadata } = stage();

    expect(
      getLandscapeLabel(
        stage({
          metadata: { ...metadata, namespace: "production-eu" },
        }),
      ),
    ).toBe("PRODUCTION-EU");
  });
});
