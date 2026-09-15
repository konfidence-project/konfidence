import type { ApiClient } from "$lib/konfidence-api/client";
import { LandscapeApiError, getLandscapes, getStages } from "$lib/landscape/api";
import type { Landscape, LandscapeResource, Stage } from "$lib/landscape/api";

type LandscapeDataStatus = "loading" | "ready" | "error";

interface RefreshOptions {
  landscapeId?: string;
}

class LandscapeDataStore {
  landscapes = $state.raw<readonly Landscape[]>([]);
  stages = $state.raw<readonly Stage[]>([]);
  status = $state<LandscapeDataStatus>("loading");
  error = $state<string>();
  errorStatus = $state<number>();
  errorSource = $state<LandscapeResource>();
  readonly #client: ApiClient;
  #controller?: AbortController;

  constructor(client: ApiClient) {
    this.#client = client;
  }

  cancel(): void {
    this.#controller?.abort();
  }

  // oxlint-disable-next-line eslint/max-statements -- Sequential loads keep request cancellation and per-endpoint state transitions together.
  async refresh(projectId: string, options: RefreshOptions = {}): Promise<void> {
    this.cancel();
    const controller = new AbortController();
    this.#controller = controller;
    this.status = "loading";
    this.error = undefined;
    this.errorStatus = undefined;
    this.errorSource = undefined;
    try {
      const landscapes = await getLandscapes({
        client: this.#client,
        projectId,
        signal: controller.signal,
      });
      const stages = await getStages({
        client: this.#client,
        landscapeId: options.landscapeId,
        projectId,
        signal: controller.signal,
      });
      if (!controller.signal.aborted) {
        this.landscapes = landscapes;
        this.stages = stages;
        this.status = "ready";
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        this.error = error instanceof Error ? error.message : "Failed to reach the API";
        this.errorStatus = error instanceof LandscapeApiError ? error.status : undefined;
        this.errorSource = error instanceof LandscapeApiError ? error.resource : undefined;
        this.status = "error";
      }
    }
  }
}

export { LandscapeDataStore };
export type { LandscapeDataStatus };
