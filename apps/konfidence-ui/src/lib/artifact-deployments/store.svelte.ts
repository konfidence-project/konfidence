import type { components } from "@konfidence/api-client/schema";
import type { ApiClient } from "$lib/konfidence-api/client";

type Landscape = components["schemas"]["Landscape"];
type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
type Stage = components["schemas"]["Stage"];
type VectorDeployment = components["schemas"]["VectorDeployment"];

type Status = "idle" | "loading" | "ready" | "error";

interface FilterQuery {
  landscapeId?: string;
  vectorDeploymentId?: string;
}

interface FetchResult {
  artifactDeployments: components["schemas"]["ArtifactDeployment"][];
  landscapes: Landscape[];
  stages: components["schemas"]["Stage"][];
  vectorDeployments: VectorDeployment[];
}

const UNAVAILABLE = "Artifact deployments are currently unavailable.";

class ArtifactDeploymentsStore {
  artifactDeployments = $state.raw<readonly ArtifactDeployment[]>([]);
  landscapes = $state.raw<readonly Landscape[]>([]);
  stages = $state.raw<readonly Stage[]>([]);
  vectorDeployments = $state.raw<readonly VectorDeployment[]>([]);
  status = $state<Status>("idle");
  error = $state<string | undefined>(undefined);
  hasLoaded = $state(false);

  readonly #client: ApiClient;
  #inflight: Promise<void> | undefined;

  constructor(client: ApiClient) {
    this.#client = client;
  }

  refresh(projectId: string, filters: FilterQuery = {}): Promise<void> {
    this.status = "loading";
    this.error = undefined;
    this.#inflight = this.#load(projectId, filters);
    return this.#inflight;
  }

  async #load(projectId: string, filters: FilterQuery): Promise<void> {
    try {
      const result = await this.#fetch(projectId, filters);
      if (result) {
        this.#applyResult(result);
      }
    } catch {
      this.#applyError(UNAVAILABLE);
    } finally {
      this.#inflight = undefined;
    }
  }

  async #fetch(projectId: string, filters: FilterQuery): Promise<FetchResult | undefined> {
    const path = { projectId };
    const [artifactRes, landscapeRes, stageRes, vectorRes] = await Promise.all([
      this.#client.GET("/v1/projects/{projectId}/artifactDeployments", {
        params: { path, query: filters },
      }),
      this.#client.GET("/v1/projects/{projectId}/landscapes", { params: { path } }),
      this.#client.GET("/v1/projects/{projectId}/stages", { params: { path } }),
      this.#client.GET("/v1/projects/{projectId}/vectorDeployments", { params: { path } }),
    ]);

    if (artifactRes.error) {
      this.#applyError(UNAVAILABLE);
      return undefined;
    }

    return {
      artifactDeployments: artifactRes.data?.data ?? [],
      landscapes: landscapeRes.data?.data ?? [],
      stages: stageRes.data?.data ?? [],
      vectorDeployments: vectorRes.data?.data ?? [],
    };
  }

  #applyResult(result: FetchResult): void {
    this.artifactDeployments = result.artifactDeployments;
    this.landscapes = result.landscapes;
    this.stages = result.stages;
    this.vectorDeployments = result.vectorDeployments;
    this.status = "ready";
    this.hasLoaded = true;
  }

  #applyError(message: string): void {
    this.status = "error";
    this.error = message;
    this.artifactDeployments = [];
    this.landscapes = [];
    this.stages = [];
    this.vectorDeployments = [];
  }
}

export { ArtifactDeploymentsStore };
