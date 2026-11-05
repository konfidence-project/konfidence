import { goto } from "$app/navigation";
import { page } from "$app/state";
import type { PromotionFilter } from "./filters.ts";
import type { Promotion } from "./promotions.ts";

const FILTER_PARAMS = ["promotionLandscape", "promotionStage", "promotionConfig"] as const;
type FilterParam = (typeof FILTER_PARAMS)[number];

class PromotionFilters {
  query = $state("");

  get selectedId(): string | undefined {
    return page.url.searchParams.get("promotion") ?? undefined;
  }

  open = (promotion: Promotion): void => {
    if (this.selectedId === promotion.id) {
      this.close();
      return;
    }
    this.updateUrl(
      (params) => params.set("promotion", promotion.id),
      this.selectedId !== undefined,
      { ...page.state, overlayOrigin: this.selectedId ? page.state.overlayOrigin : "promotion" },
    );
  };

  clearSelection = (): void => {
    if (!this.selectedId) {
      return;
    }
    this.updateUrl((params) => params.delete("promotion"), true, {
      ...page.state,
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

  get values(): PromotionFilter {
    return {
      config: page.url.searchParams.get("promotionConfig") ?? "",
      landscape: page.url.searchParams.get("promotionLandscape") ?? "",
      query: this.query,
      stage: page.url.searchParams.get("promotionStage") ?? "",
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
    this.updateUrl(
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
      { ...page.state, overlayOrigin: undefined },
    );
  };

  clear = (): void => {
    this.query = "";
    this.updateUrl(
      (params) => {
        for (const param of FILTER_PARAMS) {
          params.delete(param);
        }
      },
      this.selectedId !== undefined,
      { ...page.state, overlayOrigin: undefined },
    );
  };
}

const usePromotionFilters = (): PromotionFilters => new PromotionFilters();
export { usePromotionFilters };
