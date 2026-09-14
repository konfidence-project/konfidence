import type { ApiClient } from "$lib/konfidence-api/client";
import { LandscapeApiError, getLandscapeData } from "$lib/landscape/api";
import type { Landscape, Stage } from "$lib/landscape/api";

type LandscapeDataStatus = "loading" | "ready" | "error";

class LandscapeDataStore {
  landscapes = $state.raw<readonly Landscape[]>([]);
  stages = $state.raw<readonly Stage[]>([]);
  status = $state<LandscapeDataStatus>("loading");
  error = $state<string>();
  errorStatus = $state<number>();
  readonly #client: ApiClient;
  #controller?: AbortController;

  constructor(client: ApiClient) {
    this.#client = client;
  }

  cancel(): void {
    this.#controller?.abort();
  }

  // oxlint-disable-next-line eslint/max-statements -- Keep request cancellation and the corresponding state transitions together.
  async load(projectId: string, landscapeId?: string): Promise<void> {
    this.cancel();
    const controller = new AbortController();
    this.#controller = controller;
    this.status = "loading";
    this.error = undefined;
    this.errorStatus = undefined;
    try {
      const data = await getLandscapeData({
        client: this.#client,
        landscapeId,
        projectId,
        signal: controller.signal,
      });
      if (!controller.signal.aborted) {
        this.landscapes = data.landscapes;
        this.stages = data.stages;
        this.status = "ready";
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        this.error = error instanceof Error ? error.message : "Failed to reach the API";
        this.errorStatus = error instanceof LandscapeApiError ? error.status : undefined;
        this.status = "error";
      }
    }
  }
}

export { LandscapeDataStore };
export type { LandscapeDataStatus };
