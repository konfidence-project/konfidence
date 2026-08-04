import { getRequestEvent, query } from "$app/server";
import * as valibot from "valibot";
import { createRequestClient } from "$lib/server/konfidence-api/client";
import { throwApiError } from "$lib/server/konfidence-api/errors";

const projectIdSchema = valibot.pipe(valibot.string(), valibot.nonEmpty());

const responseData = <Data>(
  event: ReturnType<typeof getRequestEvent>,
  result: { data?: Data; response: Response },
  message: string,
): Data => {
  if (!result.data) {
    return throwApiError(event, result.response, message);
  }
  return result.data;
};

const getProjects = query(async () => {
  const event = getRequestEvent();
  const api = createRequestClient(event);
  const result = await api.GET("/projects");
  return responseData(event, result, "Failed to load projects").data;
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
    landscapes: responseData(event, landscapes, "Failed to load landscapes").data,
    stages: responseData(event, stages, "Failed to load stages").data,
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
    landscapes: responseData(event, landscapes, "Failed to load landscapes").data,
    stages: responseData(event, stages, "Failed to load stages").data,
    vectorDeployments: responseData(event, vectorDeployments, "Failed to load vector deployments")
      .data,
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
    artifactDeployments: responseData(
      event,
      artifactDeployments,
      "Failed to load artifact deployments",
    ).data,
    landscapes: responseData(event, landscapes, "Failed to load landscapes").data,
    stages: responseData(event, stages, "Failed to load stages").data,
  };
});

export { getArtifactDeployments, getProjectLandscape, getProjects, getVectorDeployments };
