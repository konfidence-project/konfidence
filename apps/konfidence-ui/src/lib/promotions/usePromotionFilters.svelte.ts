import { goto } from "$app/navigation";
import { page } from "$app/state";
import type { PromotionFilter, PromotionUrlFilters } from "./filters.ts";

const FILTER_PARAMS = ["landscapeId", "promotionStage", "promotionConfig"] as const;
type FilterParam = (typeof FILTER_PARAMS)[number];

class PromotionFilters {
  query = $state("");

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

  updateUrl(
    mutate: (params: URLSearchParams) => void,
    replace = false,
    state: App.PageState = page.state,
  ): void {
    const next = new globalThis.URL(page.url.href);
    mutate(next.searchParams);
    // eslint-disable-next-line svelte/no-navigation-without-resolve -- Preserve the current resolved URL, including embedded mode.
    void goto(next, { replace, reset: false, state });
  }

  change = (param: FilterParam, value: string): void => {
    this.updateUrl((params) => {
      if (value) {
        params.set(param, value);
      } else {
        params.delete(param);
      }
      for (const downstream of FILTER_PARAMS.slice(FILTER_PARAMS.indexOf(param) + 1)) {
        params.delete(downstream);
      }
    });
  };

  clear = (): void => {
    this.query = "";
    this.updateUrl((params) => {
      for (const param of FILTER_PARAMS) {
        params.delete(param);
      }
    });
  };
}

const usePromotionFilters = (urlFilters: () => PromotionUrlFilters): PromotionFilters =>
  new PromotionFilters(urlFilters);
export { usePromotionFilters };
