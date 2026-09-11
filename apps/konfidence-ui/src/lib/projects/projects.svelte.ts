import type { paths } from "@konfidence/api-client/schema";
import type { ApiClient } from "$lib/konfidence-api/client";
import type { Project, ProjectsStatus } from "$lib/projects/projectContext";

const PROJECTS_ROUTE = "/v1/projects" satisfies keyof paths;

class ProjectsStore {
  projects = $state.raw<readonly Project[]>([]);
  status = $state<ProjectsStatus>("idle");
  error = $state<string | undefined>(undefined);

  readonly #client: ApiClient;
  #inflight: Promise<void> | undefined;

  constructor(client: ApiClient) {
    this.#client = client;
  }

  refresh(): Promise<void> {
    if (this.#inflight) {
      return this.#inflight;
    }
    this.status = "loading";
    this.error = undefined;
    this.#inflight = this.#load();
    return this.#inflight;
  }

  async #load(): Promise<void> {
    try {
      const result = await this.#client.GET(PROJECTS_ROUTE);
      if (result.data) {
        this.projects = result.data.data;
        this.status = "ready";
      } else {
        this.#applyError(`Unable to load projects (status ${result.response.status})`);
      }
    } catch (error) {
      this.#applyError(error instanceof Error ? error.message : "Failed to reach the API");
    } finally {
      this.#inflight = undefined;
    }
  }

  #applyError(message: string): void {
    this.status = "error";
    this.error = message;
  }
}

export { ProjectsStore };
