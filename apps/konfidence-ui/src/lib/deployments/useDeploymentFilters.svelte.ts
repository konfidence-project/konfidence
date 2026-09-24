import { goto } from "$app/navigation";
import { page } from "$app/state";

import { DEEP_LINK_PARAM } from "./params.js";

/**
 * A server-side filter dimension backed by a URL search param. `value` reads
 * the currently-selected value (from the load data / URL), and `param` is the
 * search-param key used to persist a selection.
 */
interface ServerFilterDimension {
  param: string;
  value: () => string | undefined;
}

interface MatchableRow {
  id: string;
}

interface DeploymentFilterMatchers<Row extends MatchableRow, Status extends string> {
  matchesQuery: (row: Row, query: string) => boolean;
  matchesStatus: (row: Row, status: Status | "") => boolean;
}

interface UseDeploymentFiltersOptions<Row extends MatchableRow, Status extends string> {
  rows: () => readonly Row[];
  /** Server-side filter dimensions (e.g. landscape, vector deployment). */
  dimensions: readonly ServerFilterDimension[];
  matchers: DeploymentFilterMatchers<Row, Status>;
}

/**
 * Shared client-side filter + deep-link state for the deployment views.
 *
 * Handles the free-text query, status dropdown, URL-driven "detail" panel, and
 * an arbitrary set of server-side filter dimensions (each mapped to a URL
 * search param). The vector view uses a single `landscapeId` dimension; the
 * artifact view adds `vectorDeploymentId`.
 */
class DeploymentFilters<Row extends MatchableRow, Status extends string> {
  query = $state("");
  status = $state<"" | Status>("");

  readonly #rows: () => readonly Row[];
  readonly #dimensions: readonly ServerFilterDimension[];
  readonly #matchers: DeploymentFilterMatchers<Row, Status>;

  constructor(options: UseDeploymentFiltersOptions<Row, Status>) {
    this.#rows = options.rows;
    this.#dimensions = options.dimensions;
    this.#matchers = options.matchers;
  }

  get visibleRows(): Row[] {
    return this.#rows().filter(
      (row) =>
        this.#matchers.matchesQuery(row, this.query) &&
        this.#matchers.matchesStatus(row, this.status),
    );
  }

  get selectedId(): string | undefined {
    return page.url.searchParams.get(DEEP_LINK_PARAM) ?? undefined;
  }

  get selected(): Row | undefined {
    const id = this.selectedId;
    return id ? this.#rows().find((row) => row.id === id) : undefined;
  }

  get panelOpen(): boolean {
    return this.selected !== undefined;
  }

  get serverFiltersActive(): boolean {
    return this.#dimensions.some((dimension) => dimension.value() !== undefined);
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

  openRow = (row: Row): void => {
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

  /**
   * Sets a server-filter dimension to `value` (or clears it when empty). When
   * changing a dimension, all dimensions that come after it are cleared (so
   * e.g. changing the landscape resets the vector-deployment selection), along
   * with the open detail panel.
   */
  changeDimension = (param: string, value: string): void => {
    const index = this.#dimensions.findIndex((dimension) => dimension.param === param);

    this.#updateUrl((params) => {
      if (value === "") {
        params.delete(param);
      } else {
        params.set(param, value);
      }
      if (index !== -1) {
        for (const dimension of this.#dimensions.slice(index + 1)) {
          params.delete(dimension.param);
        }
      }
      params.delete(DEEP_LINK_PARAM);
    });
  };

  clearFilters = (): void => {
    this.query = "";
    this.status = "";
    if (this.serverFiltersActive || page.url.searchParams.has(DEEP_LINK_PARAM)) {
      this.#updateUrl((params) => {
        for (const dimension of this.#dimensions) {
          params.delete(dimension.param);
        }
        params.delete(DEEP_LINK_PARAM);
      });
    }
  };
}

const useDeploymentFilters = <Row extends MatchableRow, Status extends string>(
  options: UseDeploymentFiltersOptions<Row, Status>,
): DeploymentFilters<Row, Status> => new DeploymentFilters(options);

export { useDeploymentFilters };
export type { DeploymentFilters, ServerFilterDimension };
