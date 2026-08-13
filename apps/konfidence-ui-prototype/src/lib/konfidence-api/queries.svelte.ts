import api from "$lib/konfidence-api/client";
import { HTTP_UNAUTHORIZED } from "$lib/http-status";

class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
  }
}

class ApiQuery<Data> {
  current = $state.raw<Data>();
  error = $state.raw<unknown>();
  loading = $state(false);
  readonly ready = $derived(this.current !== undefined);

  constructor(private readonly load: () => Promise<Data>) {
    void this.refresh();
  }

  async refresh(): Promise<void> {
    this.loading = true;
    this.error = undefined;
    try {
      this.current = await this.load();
    } catch (error) {
      this.error = error;
    } finally {
      this.loading = false;
    }
  }
}

const responseData = <Data>(result: { data?: Data; response: Response }, message: string): Data => {
  if (result.response.status === HTTP_UNAUTHORIZED) {
    const returnTo = globalThis.location.pathname + globalThis.location.search;
    globalThis.location.assign(`/api/v1/login?return_url=${encodeURIComponent(returnTo)}`);
  }
  if (!result.data) {
    throw new ApiError(message, result.response.status);
  }
  return result.data;
};

const getProjects = async () => {
  const result = await api.GET("/v1/projects");
  return responseData(result, "Failed to load projects").data;
};

const getIdentity = async () => {
  const result = await api.GET("/v1/identity");
  return responseData(result, "Failed to load the signed-in identity");
};

const getProjectLandscape = (projectId: string) =>
  new ApiQuery(async () => {
    const params = { params: { path: { projectId } } } as const;
    const [landscapes, stages] = await Promise.all([
      api.GET("/v1/projects/{projectId}/landscapes", params),
      api.GET("/v1/projects/{projectId}/stages", params),
    ]);
    return {
      landscapes: responseData(landscapes, "Failed to load landscapes").data,
      stages: responseData(stages, "Failed to load stages").data,
    };
  });

const getVectorDeployments = (projectId: string) =>
  new ApiQuery(async () => {
    const params = { params: { path: { projectId } } } as const;
    const [landscapes, stages, vectorDeployments] = await Promise.all([
      api.GET("/v1/projects/{projectId}/landscapes", params),
      api.GET("/v1/projects/{projectId}/stages", params),
      api.GET("/v1/projects/{projectId}/vectorDeployments", params),
    ]);
    return {
      landscapes: responseData(landscapes, "Failed to load landscapes").data,
      stages: responseData(stages, "Failed to load stages").data,
      vectorDeployments: responseData(vectorDeployments, "Failed to load vector deployments").data,
    };
  });

const getArtifactDeployments = (projectId: string) =>
  new ApiQuery(async () => {
    const params = { params: { path: { projectId } } } as const;
    const [artifactDeployments, landscapes, stages] = await Promise.all([
      api.GET("/v1/projects/{projectId}/artifactDeployments", params),
      api.GET("/v1/projects/{projectId}/landscapes", params),
      api.GET("/v1/projects/{projectId}/stages", params),
    ]);
    return {
      artifactDeployments: responseData(artifactDeployments, "Failed to load artifact deployments")
        .data,
      landscapes: responseData(landscapes, "Failed to load landscapes").data,
      stages: responseData(stages, "Failed to load stages").data,
    };
  });

export {
  ApiError,
  getArtifactDeployments,
  getIdentity,
  getProjectLandscape,
  getProjects,
  getVectorDeployments,
};
