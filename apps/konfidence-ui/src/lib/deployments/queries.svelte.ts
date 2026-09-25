import { createQuery, fail, succeed } from "svelte-tiny-query";

import { getApiClient } from "$lib/konfidence-api/client-instance";
import type { components } from "@konfidence/api-client/schema";

type Landscape = components["schemas"]["Landscape"];
type Stage = components["schemas"]["Stage"];
type VectorDeployment = components["schemas"]["VectorDeployment"];
type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];

/**
 * Shared query definitions for project-scoped deployment data.
 *
 * Each resource is defined once here and reused across the vector- and
 * artifact-deployment views. `landscapes` and `stages` in particular are shared
 * by both views (and were previously fetched by hand in three separate stores),
 * so svelte-tiny-query now caches and de-dupes them across views.
 *
 * Loading functions return `fail(...)` instead of throwing so the ApiClient's
 * 401 middleware still fires on unauthorized responses.
 */

// These param types carry an index signature so they satisfy svelte-tiny-query's
// `QueryParam` constraint (a serializable key). We use the key-function form of
// `createQuery`, so the extra keys are never actually serialized.
interface ProjectScope {
  [key: string]: string | undefined;
  projectId: string;
}

interface LandscapeFilter {
  [key: string]: string | undefined;
  projectId: string;
  landscapeId?: string;
}

interface VectorDeploymentFilter {
  [key: string]: string | undefined;
  projectId: string;
  landscapeId?: string;
}

interface ArtifactDeploymentFilter {
  [key: string]: string | undefined;
  projectId: string;
  landscapeId?: string;
  vectorDeploymentId?: string;
}

const LANDSCAPES_UNAVAILABLE = "Landscapes are currently unavailable.";
const STAGES_UNAVAILABLE = "Stages are currently unavailable.";
const VECTOR_DEPLOYMENTS_UNAVAILABLE = "Vector deployments are currently unavailable.";
const ARTIFACT_DEPLOYMENTS_UNAVAILABLE = "Artifact deployments are currently unavailable.";

const useLandscapes = createQuery<string, ProjectScope, readonly Landscape[]>(
  (param) => ["projects", param.projectId, "landscapes"],
  async ({ projectId }) => {
    const { data, error } = await getApiClient().GET("/v1/projects/{projectId}/landscapes", {
      params: { path: { projectId } },
    });
    if (error) {
      return fail<string>(LANDSCAPES_UNAVAILABLE);
    }
    return succeed<readonly Landscape[]>(data?.data ?? []);
  },
);

const useStages = createQuery<string, LandscapeFilter, readonly Stage[]>(
  (param) =>
    param.landscapeId
      ? ["projects", param.projectId, "stages", param.landscapeId]
      : ["projects", param.projectId, "stages"],
  async ({ projectId, landscapeId }) => {
    const { data, error } = await getApiClient().GET("/v1/projects/{projectId}/stages", {
      params: { path: { projectId }, query: landscapeId ? { landscapeId } : {} },
    });
    if (error) {
      return fail<string>(STAGES_UNAVAILABLE);
    }
    return succeed<readonly Stage[]>(data?.data ?? []);
  },
);

const useVectorDeployments = createQuery<
  string,
  VectorDeploymentFilter,
  readonly VectorDeployment[]
>(
  (param) => ["projects", param.projectId, "vectorDeployments", param.landscapeId ?? ""],
  async ({ projectId, landscapeId }) => {
    const { data, error } = await getApiClient().GET("/v1/projects/{projectId}/vectorDeployments", {
      params: {
        path: { projectId },
        query: landscapeId ? { landscapeId } : {},
      },
    });
    if (error) {
      return fail<string>(VECTOR_DEPLOYMENTS_UNAVAILABLE);
    }
    return succeed<readonly VectorDeployment[]>(data?.data ?? []);
  },
);

const useArtifactDeployments = createQuery<
  string,
  ArtifactDeploymentFilter,
  readonly ArtifactDeployment[]
>(
  (param) => [
    "projects",
    param.projectId,
    "artifactDeployments",
    param.landscapeId ?? "",
    param.vectorDeploymentId ?? "",
  ],
  async ({ projectId, landscapeId, vectorDeploymentId }) => {
    const query: { landscapeId?: string; vectorDeploymentId?: string } = {};
    if (landscapeId) {
      query.landscapeId = landscapeId;
    }
    if (vectorDeploymentId) {
      query.vectorDeploymentId = vectorDeploymentId;
    }
    const { data, error } = await getApiClient().GET(
      "/v1/projects/{projectId}/artifactDeployments",
      {
        params: { path: { projectId }, query },
      },
    );
    if (error) {
      return fail<string>(ARTIFACT_DEPLOYMENTS_UNAVAILABLE);
    }
    return succeed<readonly ArtifactDeployment[]>(data?.data ?? []);
  },
);

export { useArtifactDeployments, useLandscapes, useStages, useVectorDeployments };
