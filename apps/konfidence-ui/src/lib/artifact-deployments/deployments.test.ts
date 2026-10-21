import { describe, expect, it } from "vitest";
import type {
  ArtifactDeployment as ApiArtifactDeployment,
  Landscape as ApiLandscape,
  Stage as ApiStage,
  VectorDeployment as ApiVectorDeployment,
} from "$lib/konfidence-api/types";
import {
  matchesQuery,
  matchesStatus,
  statusLabel,
  statusTone,
  toArtifactDeploymentRows,
} from "./deployments.js";

const REPOSITORY = "https://bp.common.repositories.cloud.sap/konfidence/delivery-vector";

const landscape = (id: string, name: string): ApiLandscape => ({ id, name });

const stage = (id: string, name: string, landscapeId: string): ApiStage => ({
  id,
  landscapeId,
  name,
});

interface VectorInput {
  id: string;
  landscapeId: string;
  stageId: string;
  componentVersion: string;
}

const vector = ({
  id,
  landscapeId,
  stageId,
  componentVersion,
}: VectorInput): ApiVectorDeployment => ({
  id,
  landscapeId,
  stageId,
  status: "DeploymentReady",
  vector: {
    componentName: "github.com/konfidence-project/example-app/vector",
    componentVersion,
    repository: REPOSITORY,
  },
});

interface DeploymentInput {
  id: string;
  landscapeId: string;
  stageIds: string[];
  vectorDeploymentIds: string[];
  componentName: string;
  componentVersion: string;
  status?: ApiArtifactDeployment["status"];
}

const deployment = ({
  id,
  landscapeId,
  stageIds,
  vectorDeploymentIds,
  componentName,
  componentVersion,
  status = "ArtifactDeployed",
}: DeploymentInput): ApiArtifactDeployment => ({
  artifact: {
    componentName,
    componentVersion,
    repository: REPOSITORY,
  },
  id,
  landscapeId,
  stageIds,
  status,
  vectorDeploymentIds,
});

const landscapes = [landscape("development", "Development"), landscape("test", "Test")];
const stages = [
  stage("dev-us30", "dev-us30", "development"),
  stage("test-eu20", "test-eu20", "test"),
];
const vectors = [
  vector({
    componentVersion: "2026.9.29-201426000Z",
    id: "dev-us30-20260929-201426000Z",
    landscapeId: "development",
    stageId: "dev-us30",
  }),
  vector({
    componentVersion: "2026.9.24-151030000Z",
    id: "test-eu20-20260924-151030000Z",
    landscapeId: "test",
    stageId: "test-eu20",
  }),
];

const build = (artifactDeployments: ApiArtifactDeployment[]) =>
  toArtifactDeploymentRows({
    artifactDeployments,
    landscapes,
    stages,
    vectorDeployments: vectors,
  });

describe("toArtifactDeploymentRows", () => {
  it("resolves landscape, stage and vector-deployment names", () => {
    const [row] = build([
      deployment({
        componentName: "payments-api",
        componentVersion: "3.4.1",
        id: "a-1",
        landscapeId: "development",
        stageIds: ["dev-us30"],
        vectorDeploymentIds: ["dev-us30-20260929-201426000Z"],
      }),
    ]);
    expect(row?.landscape).toBe("Development");
    expect(row?.stageNames).toEqual(["dev-us30"]);
    expect(row?.vectorDeploymentLabels).toEqual([
      "github.com/konfidence-project/example-app/vector@2026.9.29-201426000Z",
    ]);
    expect(row?.relatedVectorDeployments).toHaveLength(1);
    expect(row?.repository).toBe(REPOSITORY);
    expect(row?.component).toBe("payments-api");
    expect(row?.version).toBe("3.4.1");
  });

  it("falls back to raw ids when a lookup misses", () => {
    const [row] = build([
      deployment({
        componentName: "payments-ui",
        componentVersion: "2.7.0",
        id: "a-2",
        landscapeId: "ghost",
        stageIds: ["ghost-stage"],
        vectorDeploymentIds: ["ghost-vector"],
      }),
    ]);
    expect(row?.landscape).toBe("ghost");
    expect(row?.stageNames).toEqual(["ghost-stage"]);
    expect(row?.vectorDeploymentLabels).toEqual(["ghost-vector"]);
    expect(row?.relatedVectorDeployments).toHaveLength(0);
  });

  it("preserves the raw API payload for the detail view", () => {
    const source = deployment({
      componentName: "payments-api",
      componentVersion: "3.4.1",
      id: "a-3",
      landscapeId: "development",
      stageIds: ["dev-us30"],
      status: "ArtifactFetched",
      vectorDeploymentIds: ["dev-us30-20260929-201426000Z"],
    });
    const [row] = build([source]);
    expect(row?.raw).toBe(source);
    expect(row?.status).toBe("ArtifactFetched");
  });

  it("emits an empty list when the API returns nothing", () => {
    expect(build([])).toEqual([]);
  });
});

describe("matchesQuery", () => {
  const [row] = build([
    deployment({
      componentName: "payments-api",
      componentVersion: "3.4.1",
      id: "dev-us30-20260929-201426000Z-payments-api",
      landscapeId: "development",
      stageIds: ["dev-us30"],
      vectorDeploymentIds: ["dev-us30-20260929-201426000Z"],
    }),
  ]);

  it("returns true for an empty query", () => {
    expect(matchesQuery(row!, "")).toBe(true);
    expect(matchesQuery(row!, "   ")).toBe(true);
  });

  it("matches component name case-insensitively", () => {
    expect(matchesQuery(row!, "PAYMENTS-API")).toBe(true);
  });

  it("matches version, repository, landscape, stage and vector-deployment label", () => {
    expect(matchesQuery(row!, "3.4.1")).toBe(true);
    expect(matchesQuery(row!, "bp.common.repositories.cloud.sap")).toBe(true);
    expect(matchesQuery(row!, "development")).toBe(true);
    expect(matchesQuery(row!, "dev-us30")).toBe(true);
    expect(matchesQuery(row!, "github.com/konfidence-project/example-app/vector")).toBe(true);
  });

  it("matches the deployment id", () => {
    expect(matchesQuery(row!, "dev-us30-20260929-201426000Z-payments-api")).toBe(true);
  });

  it("returns false when no field contains the query", () => {
    expect(matchesQuery(row!, "not-there")).toBe(false);
  });
});

describe("matchesStatus", () => {
  const [deployed] = build([
    deployment({
      componentName: "x",
      componentVersion: "1",
      id: "a",
      landscapeId: "development",
      stageIds: [],
      status: "ArtifactDeployed",
      vectorDeploymentIds: [],
    }),
  ]);
  const [fetched] = build([
    deployment({
      componentName: "x",
      componentVersion: "1",
      id: "b",
      landscapeId: "development",
      stageIds: [],
      status: "ArtifactFetched",
      vectorDeploymentIds: [],
    }),
  ]);

  it("accepts everything when the filter is empty", () => {
    expect(matchesStatus(deployed!, "")).toBe(true);
    expect(matchesStatus(fetched!, "")).toBe(true);
  });

  it("filters by exact status match", () => {
    expect(matchesStatus(deployed!, "ArtifactDeployed")).toBe(true);
    expect(matchesStatus(deployed!, "ArtifactFetched")).toBe(false);
    expect(matchesStatus(fetched!, "ArtifactFetched")).toBe(true);
  });
});

describe("status label helpers", () => {
  it("maps API statuses to human labels and design-system tones", () => {
    expect(statusLabel("ArtifactFetched")).toBe("Fetched");
    expect(statusLabel("ArtifactDeployed")).toBe("Deployed");
    expect(statusLabel("AppHealthy")).toBe("Healthy");
    expect(statusLabel("Ready")).toBe("Ready");
    expect(statusTone("ArtifactFetched")).toBe("deploying");
    expect(statusTone("ArtifactDeployed")).toBe("healthy");
    expect(statusTone("AppHealthy")).toBe("healthy");
    expect(statusTone("Ready")).toBe("healthy");
  });
});
