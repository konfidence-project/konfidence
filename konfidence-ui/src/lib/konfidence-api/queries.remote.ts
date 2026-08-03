import { getRequestEvent, query } from "$app/server";
import * as valibot from "valibot";
import { createRequestClient } from "$lib/server/konfidence-api/client";
import { type ApiErrorOptions, throwApiError } from "$lib/server/konfidence-api/errors";

const projectIdSchema = valibot.pipe(valibot.string(), valibot.nonEmpty());

const responseData = <Data>(
  event: ReturnType<typeof getRequestEvent>,
  result: { data?: Data; response: Response },
  options: ApiErrorOptions,
): Data => {
  if (!result.data) {
    return throwApiError(event, result.response, options);
  }
  return result.data;
};

const PROJECTS_ERROR: ApiErrorOptions = {
  code: "FAILED_LOAD_PROJECTS",
  message: "Failed to load projects",
};
const LANDSCAPES_ERROR: ApiErrorOptions = {
  code: "FAILED_LOAD_LANDSCAPES",
  message: "Failed to load landscapes",
};
const STAGES_ERROR: ApiErrorOptions = {
  code: "FAILED_LOAD_STAGES",
  message: "Failed to load stages",
};
const VECTOR_DEPLOYMENTS_ERROR: ApiErrorOptions = {
  code: "FAILED_LOAD_VECTOR_DEPLOYMENTS",
  message: "Failed to load vector deployments",
};
const ARTIFACT_DEPLOYMENTS_ERROR: ApiErrorOptions = {
  code: "FAILED_LOAD_ARTIFACT_DEPLOYMENTS",
  message: "Failed to load artifact deployments",
};

const getProjects = query(async () => {
  const event = getRequestEvent();
  const api = createRequestClient(event);
  const result = await api.GET("/projects");
  return responseData(event, result, PROJECTS_ERROR).data;
});

const getProjectLandscape = query(projectIdSchema, async (projectId) => {
  const event = getRequestEvent();
  const api = createRequestClient(event);
  const params = { params: { path: { projectId } } } as const;
  const [landscapes, stages] = await Promise.all([
    api.GET("/projects/{projectId}/landscapes", params),
    api.GET("/projects/{projectId}/stages", params),
  ]);
  return {
    landscapes: responseData(event, landscapes, LANDSCAPES_ERROR).data,
    stages: responseData(event, stages, STAGES_ERROR).data,
  };
});

const getVectorDeployments = query(projectIdSchema, async (projectId) => {
  const event = getRequestEvent();
  const api = createRequestClient(event);
  const params = { params: { path: { projectId } } } as const;
  const [landscapes, stages, vectorDeployments] = await Promise.all([
    api.GET("/projects/{projectId}/landscapes", params),
    api.GET("/projects/{projectId}/stages", params),
    api.GET("/projects/{projectId}/vectorDeployments", params),
  ]);
  return {
    landscapes: responseData(event, landscapes, LANDSCAPES_ERROR).data,
    stages: responseData(event, stages, STAGES_ERROR).data,
    vectorDeployments: responseData(event, vectorDeployments, VECTOR_DEPLOYMENTS_ERROR).data,
  };
});

const getArtifactDeployments = query(projectIdSchema, async (projectId) => {
  const event = getRequestEvent();
  const api = createRequestClient(event);
  const params = { params: { path: { projectId } } } as const;
  const [artifactDeployments, landscapes, stages] = await Promise.all([
    api.GET("/projects/{projectId}/artifactDeployments", params),
    api.GET("/projects/{projectId}/landscapes", params),
    api.GET("/projects/{projectId}/stages", params),
  ]);
  return {
    artifactDeployments: responseData(event, artifactDeployments, ARTIFACT_DEPLOYMENTS_ERROR).data,
    landscapes: responseData(event, landscapes, LANDSCAPES_ERROR).data,
    stages: responseData(event, stages, STAGES_ERROR).data,
  };
});

export { getArtifactDeployments, getProjectLandscape, getProjects, getVectorDeployments };
