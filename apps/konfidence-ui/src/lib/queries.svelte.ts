import { createQuery, fail, succeed } from "svelte-tiny-query";

import { getApiClient } from "$lib/konfidence-api/client-instance";
import type {
  ArtifactDeployment,
  Landscape,
  Project,
  Stage,
  VectorDeployment,
} from "$lib/konfidence-api/types";

interface ProjectScope {
  [key: string]: string | undefined;
  projectId: string;
}

interface QueryError {
  message: string;
  status?: number;
}

interface LandscapeFilter {
  [key: string]: string | undefined;
  projectId: string;
  landscapeId?: string;
}

interface ArtifactDeploymentFilter extends LandscapeFilter {
  vectorDeploymentId?: string;
}

// fail(...) populates the query error state for API failures.
const LANDSCAPES_UNAVAILABLE = "Landscapes are currently unavailable.";
const PROJECTS_UNAVAILABLE = "Projects are currently unavailable.";
const STAGES_UNAVAILABLE = "Stages are currently unavailable.";
const VECTOR_DEPLOYMENTS_UNAVAILABLE = "Vector deployments are currently unavailable.";
const ARTIFACT_DEPLOYMENTS_UNAVAILABLE = "Artifact deployments are currently unavailable.";

const useProjects = createQuery<QueryError, string, readonly Project[]>(
  (email) => ["projects", email],
  async (_email, signal) => {
    try {
      const { data, error, response } = await getApiClient().GET("/v1/projects", { signal });
      if (error || !response.ok || !data?.data) {
        return fail<QueryError>({ message: PROJECTS_UNAVAILABLE, status: response.status });
      }
      return succeed<readonly Project[]>(data.data);
    } catch {
      return fail<QueryError>({ message: PROJECTS_UNAVAILABLE });
    }
  },
);

const useLandscapes = createQuery<QueryError, ProjectScope, readonly Landscape[]>(
  (param) => ["projects", param.projectId, "landscapes"],
  async ({ projectId }, signal) => {
    try {
      const { data, error, response } = await getApiClient().GET(
        "/v1/projects/{projectId}/landscapes",
        {
          params: { path: { projectId } },
          signal,
        },
      );
      if (error || !response.ok || !data?.data) {
        return fail<QueryError>({ message: LANDSCAPES_UNAVAILABLE, status: response.status });
      }
      return succeed<readonly Landscape[]>(data.data);
    } catch {
      return fail<QueryError>({ message: LANDSCAPES_UNAVAILABLE });
    }
  },
);

const useStages = createQuery<QueryError, LandscapeFilter, readonly Stage[]>(
  (param) =>
    param.landscapeId
      ? ["projects", param.projectId, "stages", param.landscapeId]
      : ["projects", param.projectId, "stages"],
  async ({ projectId, landscapeId }, signal) => {
    try {
      const { data, error, response } = await getApiClient().GET(
        "/v1/projects/{projectId}/stages",
        {
          params: { path: { projectId }, query: landscapeId ? { landscapeId } : {} },
          signal,
        },
      );
      if (error || !response.ok || !data?.data) {
        return fail<QueryError>({ message: STAGES_UNAVAILABLE, status: response.status });
      }
      return succeed<readonly Stage[]>(data.data);
    } catch {
      return fail<QueryError>({ message: STAGES_UNAVAILABLE });
    }
  },
);

const useVectorDeployments = createQuery<QueryError, LandscapeFilter, readonly VectorDeployment[]>(
  (param) => ["projects", param.projectId, "vectorDeployments", param.landscapeId ?? ""],
  async ({ projectId, landscapeId }, signal) => {
    try {
      const { data, error, response } = await getApiClient().GET(
        "/v1/projects/{projectId}/vectorDeployments",
        {
          params: {
            path: { projectId },
            query: landscapeId ? { landscapeId } : {},
          },
          signal,
        },
      );
      if (error || !response.ok || !data?.data) {
        return fail<QueryError>({
          message: VECTOR_DEPLOYMENTS_UNAVAILABLE,
          status: response.status,
        });
      }
      return succeed<readonly VectorDeployment[]>(data.data);
    } catch {
      return fail<QueryError>({ message: VECTOR_DEPLOYMENTS_UNAVAILABLE });
    }
  },
);

const useArtifactDeployments = createQuery<
  QueryError,
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
  async ({ projectId, landscapeId, vectorDeploymentId }, signal) => {
    const query = {
      ...(landscapeId ? { landscapeId } : {}),
      ...(vectorDeploymentId ? { vectorDeploymentId } : {}),
    };
    try {
      const { data, error, response } = await getApiClient().GET(
        "/v1/projects/{projectId}/artifactDeployments",
        {
          params: { path: { projectId }, query },
          signal,
        },
      );
      if (error || !response.ok || !data?.data) {
        return fail<QueryError>({
          message: ARTIFACT_DEPLOYMENTS_UNAVAILABLE,
          status: response.status,
        });
      }
      return succeed<readonly ArtifactDeployment[]>(data.data);
    } catch {
      return fail<QueryError>({ message: ARTIFACT_DEPLOYMENTS_UNAVAILABLE });
    }
  },
);

export { useArtifactDeployments, useLandscapes, useProjects, useStages, useVectorDeployments };
