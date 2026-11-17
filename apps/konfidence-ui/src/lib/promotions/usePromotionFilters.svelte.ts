import { page } from "$app/state";
import { UrlNavigation } from "#lib/urlNavigation.svelte.ts";
import type { PromotionFilter, PromotionUrlFilters } from "./filters.ts";

const FILTER_PARAMS = ["landscapeId", "promotionStage", "promotionConfig"] as const;
type FilterParam = (typeof FILTER_PARAMS)[number];

class PromotionFilters {
  query = $state("");
  private navigation = new UrlNavigation();

  get selectedId(): string | undefined {
    return page.url.searchParams.get("promotion") ?? undefined;
  }

  get selectionPending(): boolean {
    return this.navigation.pending && this.navigation.url.searchParams.has("promotion");
  }

  select = (id: string | undefined): void => {
    if (id === undefined) {
      this.clearSelection();
      return;
    }
    const currentId = this.navigation.url.searchParams.get("promotion") ?? undefined;
    if (currentId === id) {
      return;
    }
    this.navigation.update(
      (params) => params.set("promotion", id),
      this.navigation.replace ?? this.selectedId !== undefined,
      {
        ...this.navigation.state,
        overlayOrigin: currentId ? this.navigation.state.overlayOrigin : "promotion",
      },
    );
  };

  clearSelection = (excludedId?: string): void => {
    const requestedParams = this.navigation.url.searchParams;
    // Cleanup validated the committed selection. Do not erase a newer request.
    if (
      !requestedParams.has("promotion") ||
      (excludedId !== undefined && requestedParams.get("promotion") !== excludedId)
    ) {
      return;
    }
    this.navigation.update((params) => params.delete("promotion"), true, {
      ...this.navigation.state,
      overlayOrigin: undefined,
    });
  };

  close = (): void => {
    if (!this.selectedId) {
      return;
    }
    if (page.state.overlayOrigin === "promotion") {
      globalThis.history.back();
    } else {
      this.clearSelection();
    }
  };

  readonly #urlFilters: () => PromotionUrlFilters;

  constructor(urlFilters: () => PromotionUrlFilters) {
    this.#urlFilters = urlFilters;
  }

  get values(): PromotionFilter {
    return {
      ...this.#urlFilters(),
      query: this.query,
    };
  }

  get active(): boolean {
    return Object.values(this.values).some((value) => value.trim() !== "");
  }

  change = (param: FilterParam, value: string): void => {
    this.navigation.update(
      (params) => {
        if (value) {
          params.set(param, value);
        } else {
          params.delete(param);
        }
        for (const downstream of FILTER_PARAMS.slice(FILTER_PARAMS.indexOf(param) + 1)) {
          params.delete(downstream);
        }
      },
      this.selectedId !== undefined,
      { ...this.navigation.state, overlayOrigin: undefined },
    );
  };

  clear = (): void => {
    this.query = "";
    this.navigation.update(
      (params) => {
        for (const param of FILTER_PARAMS) {
          params.delete(param);
        }
      },
      this.selectedId !== undefined,
      { ...this.navigation.state, overlayOrigin: undefined },
    );
  };
}

const usePromotionFilters = (urlFilters: () => PromotionUrlFilters): PromotionFilters =>
  new PromotionFilters(urlFilters);
export { usePromotionFilters };
