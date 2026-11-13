import { goto } from "$app/navigation";
import { page } from "$app/state";
import type { PromotionFilter } from "./filters.ts";

const FILTER_PARAMS = ["promotionLandscape", "promotionStage", "promotionConfig"] as const;
type FilterParam = (typeof FILTER_PARAMS)[number];

class PromotionFilters {
  query = $state("");
  // Navigation commits asynchronously. Keep the latest requested URL, not a second
  // rendered selection, so overlapping requests compose and retain one history entry.
  private pending: { url: URL; state: App.PageState; replace: boolean } | undefined;

  get selectedId(): string | undefined {
    return page.url.searchParams.get("promotion") ?? undefined;
  }

  get selectionPending(): boolean {
    return this.pending?.url.searchParams.has("promotion") ?? false;
  }

  select = (id: string | undefined): void => {
    if (id === undefined) {
      this.clearSelection();
      return;
    }
    const currentId = this.pending
      ? (this.pending.url.searchParams.get("promotion") ?? undefined)
      : this.selectedId;
    if (currentId === id) {
      return;
    }
    this.updateUrl(
      (params) => params.set("promotion", id),
      this.pending?.replace ?? this.selectedId !== undefined,
      {
        ...(this.pending?.state ?? page.state),
        overlayOrigin: currentId ? (this.pending?.state ?? page.state).overlayOrigin : "promotion",
      },
    );
  };

  clearSelection = (excludedId?: string): void => {
    const requestedParams = (this.pending?.url ?? page.url).searchParams;
    // Cleanup validated the committed selection. Do not erase a newer request.
    if (
      !requestedParams.has("promotion") ||
      (excludedId !== undefined && requestedParams.get("promotion") !== excludedId)
    ) {
      return;
    }
    this.updateUrl((params) => params.delete("promotion"), true, {
      ...(this.pending?.state ?? page.state),
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
    state: App.PageState = this.pending?.state ?? page.state,
  ): void {
    const next = new globalThis.URL((this.pending?.url ?? page.url).href);
    mutate(next.searchParams);
    const request = { replace, state, url: next };
    this.pending = request;
    // eslint-disable-next-line svelte/no-navigation-without-resolve -- Preserve the current resolved URL, including embedded mode.
    void goto(next, { replace, reset: false, state }).finally(() => {
      if (this.pending === request) {
        this.pending = undefined;
      }
    });
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
      { ...(this.pending?.state ?? page.state), overlayOrigin: undefined },
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
      { ...(this.pending?.state ?? page.state), overlayOrigin: undefined },
    );
  };
}

const usePromotionFilters = (): PromotionFilters => new PromotionFilters();
export { usePromotionFilters };
