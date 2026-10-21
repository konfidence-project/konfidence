import { describe, expect, it } from "vitest";
import type {
  ArtifactDeployment as ApiArtifactDeployment,
  Landscape as ApiLandscape,
  Stage as ApiStage,
  VectorDeployment as ApiVectorDeployment,
} from "#lib/konfidence-api/types.ts";
import {
  matchesQuery,
  matchesStatus,
  statusLabel,
  statusTone,
  toVectorDeploymentRows,
} from "./deployments.js";

const REPOSITORY = "https://bp.common.repositories.cloud.sap/konfidence/delivery-vector";

const landscapes: ApiLandscape[] = [
  { id: "development", name: "Development" },
  { id: "test", name: "Test" },
];

const stages: ApiStage[] = [
  { id: "dev-us30", landscapeId: "development", name: "Development US" },
  { id: "test-eu20", landscapeId: "test", name: "Test EU" },
];

interface VectorInput {
  componentName?: string;
  componentVersion?: string;
  id: string;
  landscapeId: string;
  stageId: string;
  status?: ApiVectorDeployment["status"];
}

const vector = ({
  componentName = "github.com/konfidence-project/example-app/vector",
  componentVersion = "2026.9.29-201426000Z",
  id,
  landscapeId,
  stageId,
  status = "DeploymentReady",
}: VectorInput): ApiVectorDeployment => ({
  id,
  landscapeId,
  stageId,
  status,
  vector: { componentName, componentVersion, repository: REPOSITORY },
});

const artifact = (id: string, vectorDeploymentIds: string[]): ApiArtifactDeployment => ({
  artifact: {
    componentName: "payments-api",
    componentVersion: "3.4.1",
    repository: REPOSITORY,
  },
  id,
  landscapeId: "development",
  stageIds: ["dev-us30"],
  status: "ArtifactDeployed",
  vectorDeploymentIds,
});

const build = (
  vectorDeployments: ApiVectorDeployment[],
  artifactDeployments: ApiArtifactDeployment[] = [],
) => toVectorDeploymentRows({ artifactDeployments, landscapes, stages, vectorDeployments });

describe("toVectorDeploymentRows", () => {
  it("resolves delivery context and exposes vector metadata", () => {
    const source = vector({
      id: "dev-us30-20260929-201426000Z",
      landscapeId: "development",
      stageId: "dev-us30",
    });
    const [row] = build([source]);

    expect(row).toMatchObject({
      component: "github.com/konfidence-project/example-app/vector",
      id: "dev-us30-20260929-201426000Z",
      landscape: "Development",
      landscapeId: "development",
      repository: REPOSITORY,
      stageId: "dev-us30",
      stageName: "Development US",
      status: "DeploymentReady",
      version: "2026.9.29-201426000Z",
    });
    expect(row?.raw).toBe(source);
  });

  it("attaches only artifacts referencing the vector deployment", () => {
    const source = vector({
      id: "dev-us30-20260929-201426000Z",
      landscapeId: "development",
      stageId: "dev-us30",
    });
    const related = artifact("artifact-related", [source.id]);
    const shared = artifact("artifact-shared", ["another-vector", source.id]);
    const unrelated = artifact("artifact-unrelated", ["another-vector"]);

    const [row] = build([source], [related, shared, unrelated]);

    expect(row?.relatedArtifactDeployments).toEqual([related, shared]);
  });

  it("falls back to raw landscape and stage ids when lookups miss", () => {
    const [row] = build([
      vector({ id: "vector-unknown", landscapeId: "unknown", stageId: "unknown-stage" }),
    ]);

    expect(row?.landscape).toBe("unknown");
    expect(row?.stageName).toBe("unknown-stage");
  });

  it("emits an empty list when the API returns no vector deployments", () => {
    expect(build([])).toEqual([]);
  });
});

describe("matchesQuery", () => {
  const [row] = build([
    vector({
      componentName: "payments-delivery",
      componentVersion: "3.4.1",
      id: "dev-us30-20260929-201426000Z",
      landscapeId: "development",
      stageId: "dev-us30",
      status: "DeployingVector",
    }),
  ]);

  it("accepts an empty query", () => {
    expect(matchesQuery(row!, "")).toBe(true);
    expect(matchesQuery(row!, "   ")).toBe(true);
  });

  it.each([
    "dev-us30-20260929-201426000Z",
    "PAYMENTS-DELIVERY",
    "3.4.1",
    "bp.common.repositories.cloud.sap",
    "development",
    "development us",
    "deploying",
  ])("matches displayed field %s case-insensitively", (query) => {
    expect(matchesQuery(row!, query)).toBe(true);
  });

  it("rejects a query absent from every displayed field", () => {
    expect(matchesQuery(row!, "not-there")).toBe(false);
  });
});

describe("matchesStatus", () => {
  const [row] = build([
    vector({
      id: "dev-us30-20260929-201426000Z",
      landscapeId: "development",
      stageId: "dev-us30",
      status: "DeploymentReady",
    }),
  ]);

  it("accepts every status when the filter is empty", () => {
    expect(matchesStatus(row!, "")).toBe(true);
  });

  it("requires an exact status match", () => {
    expect(matchesStatus(row!, "DeploymentReady")).toBe(true);
    expect(matchesStatus(row!, "DeployingVector")).toBe(false);
    expect(matchesStatus(row!, "DeploymentFailed")).toBe(false);
  });
});

describe("status helpers", () => {
  it("maps API statuses to labels and design-system tones", () => {
    expect(statusLabel("DeployingVector")).toBe("Deploying");
    expect(statusLabel("DeploymentReady")).toBe("Ready");
    expect(statusLabel("DeploymentFailed")).toBe("Failed");
    expect(statusTone("DeployingVector")).toBe("deploying");
    expect(statusTone("DeploymentReady")).toBe("healthy");
    expect(statusTone("DeploymentFailed")).toBe("error");
  });
});
