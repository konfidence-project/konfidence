import { goto } from "$app/navigation";
import { page } from "$app/state";

import type { ArtifactDeploymentRow, ArtifactDeploymentStatus } from "./deployments.js";
import { matchesQuery, matchesStatus } from "./deployments.js";
import { DEEP_LINK_PARAM, LANDSCAPE_PARAM, VECTOR_DEPLOYMENT_PARAM } from "./params.js";

interface UseDeploymentFiltersOptions {
  rows: () => readonly ArtifactDeploymentRow[];
  selectedLandscapeId: () => string | undefined;
  selectedVectorDeploymentId: () => string | undefined;
}

class DeploymentFilters {
  query = $state("");
  status = $state<"" | ArtifactDeploymentStatus>("");

  readonly #rows: () => readonly ArtifactDeploymentRow[];
  readonly #landscapeId: () => string | undefined;
  readonly #vectorDeploymentId: () => string | undefined;

  constructor(options: UseDeploymentFiltersOptions) {
    this.#rows = options.rows;
    this.#landscapeId = options.selectedLandscapeId;
    this.#vectorDeploymentId = options.selectedVectorDeploymentId;
  }

  get visibleRows(): ArtifactDeploymentRow[] {
    return this.#rows().filter(
      (row) => matchesQuery(row, this.query) && matchesStatus(row, this.status),
    );
  }

  get selectedId(): string | undefined {
    return page.url.searchParams.get(DEEP_LINK_PARAM) ?? undefined;
  }

  get selected(): ArtifactDeploymentRow | undefined {
    const id = this.selectedId;
    return id ? this.#rows().find((row) => row.id === id) : undefined;
  }

  get panelOpen(): boolean {
    return this.selected !== undefined;
  }

  get serverFiltersActive(): boolean {
    return this.#landscapeId() !== undefined || this.#vectorDeploymentId() !== undefined;
  }

  get clientFiltersActive(): boolean {
    return this.query.trim() !== "" || this.status !== "";
  }

  get anyFiltersActive(): boolean {
    return this.serverFiltersActive || this.clientFiltersActive;
  }

  #updateUrl(mutate: (params: URLSearchParams) => void): void {
    const next = new globalThis.URL(page.url);
    mutate(next.searchParams);
    // eslint-disable-next-line svelte/no-navigation-without-resolve -- reusing the current page URL.
    void goto(next, { keepFocus: true, noScroll: true });
  }

  openRow = (row: ArtifactDeploymentRow): void => {
    if (page.url.searchParams.get(DEEP_LINK_PARAM) === row.id) {
      return;
    }
    this.#updateUrl((params) => params.set(DEEP_LINK_PARAM, row.id));
  };

  closePanel = (): void => {
    if (!page.url.searchParams.has(DEEP_LINK_PARAM)) {
      return;
    }
    this.#updateUrl((params) => params.delete(DEEP_LINK_PARAM));
  };

  changeLandscape = (value: string): void => {
    this.#updateUrl((params) => {
      if (value === "") {
        params.delete(LANDSCAPE_PARAM);
      } else {
        params.set(LANDSCAPE_PARAM, value);
      }
      params.delete(VECTOR_DEPLOYMENT_PARAM);
      params.delete(DEEP_LINK_PARAM);
    });
  };

  changeVectorDeployment = (value: string): void => {
    this.#updateUrl((params) => {
      if (value === "") {
        params.delete(VECTOR_DEPLOYMENT_PARAM);
      } else {
        params.set(VECTOR_DEPLOYMENT_PARAM, value);
      }
      params.delete(DEEP_LINK_PARAM);
    });
  };

  clearFilters = (): void => {
    this.query = "";
    this.status = "";
    if (this.serverFiltersActive || page.url.searchParams.has(DEEP_LINK_PARAM)) {
      this.#updateUrl((params) => {
        params.delete(LANDSCAPE_PARAM);
        params.delete(VECTOR_DEPLOYMENT_PARAM);
        params.delete(DEEP_LINK_PARAM);
      });
    }
  };
}

const useDeploymentFilters = (options: UseDeploymentFiltersOptions): DeploymentFilters =>
  new DeploymentFilters(options);

export { useDeploymentFilters };
export type { DeploymentFilters };
