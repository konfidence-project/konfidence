import { beforeEach, describe, expect, it, vi } from "vitest";
import { HTTP_INTERNAL_SERVER_ERROR, HTTP_NOT_FOUND, HTTP_OK } from "$lib/http-status";
import type { Stage, VectorDeployment } from "$lib/konfidence-api/types";

const getMock = vi.fn();

vi.mock("$lib/konfidence-api/client-instance", () => ({
  getApiClient: () => ({ GET: getMock }),
}));

const runQuery = async <TData, TError>(
  invoke: () => { loading: boolean; data: TData | undefined; error: TError | undefined },
): Promise<{ data: TData | undefined; error: TError | undefined }> => {
  let state: { loading: boolean; data: TData | undefined; error: TError | undefined } | undefined =
    undefined;
  const cleanup = $effect.root(() => {
    state = invoke();
    $effect(() => {
      void state!.loading;
      void state!.data;
      void state!.error;
    });
  });

  await vi.waitFor(() => {
    expect(state!.loading).toBe(false);
  });

  const settled = { data: state!.data, error: state!.error };
  cleanup();
  return settled;
};

const vectorDeployment: VectorDeployment = {
  id: "vd-1",
  landscapeId: "ls-1",
  stageId: "st-1",
  status: "DeploymentReady",
  vector: { componentName: "comp", componentVersion: "1.0.0", repository: "repo" },
};

// A distinct vectorDeploymentId per test keeps the module-level query cache from leaking between cases.
let nextId = 0;
const freshParam = (): { landscapeId: string; projectId: string; vectorDeploymentId: string } => ({
  landscapeId: "ls-1",
  projectId: "proj-1",
  vectorDeploymentId: `vd-${(nextId += 1)}`,
});

beforeEach(() => {
  getMock.mockReset();
});

describe("useVectorDeployment", () => {
  it("returns the single vector deployment on success (data is unwrapped, not data.data)", async () => {
    getMock.mockResolvedValue({
      data: vectorDeployment,
      response: { ok: true, status: HTTP_OK },
    });
    const { useVectorDeployment } = await import("$lib/queries.svelte");

    const result = await runQuery(() => useVectorDeployment(() => freshParam()));

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual(vectorDeployment);
  });

  it("calls the single vector deployment endpoint with the path params", async () => {
    getMock.mockResolvedValue({
      data: vectorDeployment,
      response: { ok: true, status: HTTP_OK },
    });
    const { useVectorDeployment } = await import("$lib/queries.svelte");
    const param = freshParam();

    await runQuery(() => useVectorDeployment(() => param));

    expect(getMock).toHaveBeenCalledWith(
      "/v1/projects/{projectId}/landscapes/{landscapeId}/vectorDeployments/{vectorDeploymentId}",
      expect.objectContaining({
        params: {
          path: {
            landscapeId: param.landscapeId,
            projectId: param.projectId,
            vectorDeploymentId: param.vectorDeploymentId,
          },
        },
      }),
    );
  });

  it("surfaces a generic error with the response status on non-404 failures", async () => {
    getMock.mockResolvedValue({
      data: undefined,
      error: { message: "boom" },
      response: { ok: false, status: HTTP_INTERNAL_SERVER_ERROR },
    });
    const { useVectorDeployment } = await import("$lib/queries.svelte");

    const result = await runQuery(() => useVectorDeployment(() => freshParam()));

    expect(result.data).toBeUndefined();
    expect(result.error?.status).toBe(HTTP_INTERNAL_SERVER_ERROR);
    expect(result.error?.message).toBeDefined();
  });

  it("carries a 404 status so callers can distinguish a not-found outcome", async () => {
    getMock.mockResolvedValue({
      data: undefined,
      error: { message: "not found" },
      response: { ok: false, status: HTTP_NOT_FOUND },
    });
    const { useVectorDeployment } = await import("$lib/queries.svelte");

    const result = await runQuery(() => useVectorDeployment(() => freshParam()));

    expect(result.data).toBeUndefined();
    expect(result.error?.status).toBe(HTTP_NOT_FOUND);
  });
});

const stage: Stage = {
  activeStageVersion: undefined,
  id: "st-1",
  landscapeId: "ls-1",
  name: "dev-api",
  targetStageVersion: undefined,
};

let nextStageId = 0;
const freshStageParam = (): { landscapeId: string; projectId: string; stageId: string } => ({
  landscapeId: "ls-1",
  projectId: "proj-1",
  stageId: `st-${(nextStageId += 1)}`,
});

describe("useStage", () => {
  it("returns the single stage on success (data is unwrapped, not data.data)", async () => {
    getMock.mockResolvedValue({
      data: stage,
      response: { ok: true, status: HTTP_OK },
    });
    const { useStage } = await import("$lib/queries.svelte");

    const result = await runQuery(() => useStage(() => freshStageParam()));

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual(stage);
  });

  it("calls the single stage endpoint with the path params", async () => {
    getMock.mockResolvedValue({
      data: stage,
      response: { ok: true, status: HTTP_OK },
    });
    const { useStage } = await import("$lib/queries.svelte");
    const param = freshStageParam();

    await runQuery(() => useStage(() => param));

    expect(getMock).toHaveBeenCalledWith(
      "/v1/projects/{projectId}/landscapes/{landscapeId}/stages/{stageId}",
      expect.objectContaining({
        params: {
          path: {
            landscapeId: param.landscapeId,
            projectId: param.projectId,
            stageId: param.stageId,
          },
        },
      }),
    );
  });

  it("surfaces a generic error with the response status on non-404 failures", async () => {
    getMock.mockResolvedValue({
      data: undefined,
      error: { message: "boom" },
      response: { ok: false, status: HTTP_INTERNAL_SERVER_ERROR },
    });
    const { useStage } = await import("$lib/queries.svelte");

    const result = await runQuery(() => useStage(() => freshStageParam()));

    expect(result.data).toBeUndefined();
    expect(result.error?.status).toBe(HTTP_INTERNAL_SERVER_ERROR);
    expect(result.error?.message).toBeDefined();
  });

  it("carries a 404 status so callers can distinguish a not-found outcome", async () => {
    getMock.mockResolvedValue({
      data: undefined,
      error: { message: "not found" },
      response: { ok: false, status: HTTP_NOT_FOUND },
    });
    const { useStage } = await import("$lib/queries.svelte");

    const result = await runQuery(() => useStage(() => freshStageParam()));

    expect(result.data).toBeUndefined();
    expect(result.error?.status).toBe(HTTP_NOT_FOUND);
  });
});
