import { goto } from "$app/navigation";
import { page } from "$app/state";

/** Composes overlapping URL edits before SvelteKit commits the latest navigation. */
class UrlNavigation {
  #pending = $state.raw<{ url: URL; state: App.PageState; replace: boolean }>();

  get url(): URL | typeof page.url {
    return this.#pending?.url ?? page.url;
  }

  get state(): App.PageState {
    return this.#pending?.state ?? page.state;
  }

  get replace(): boolean | undefined {
    return this.#pending?.replace;
  }

  get pending(): boolean {
    return this.#pending !== undefined;
  }

  update(
    mutate: (params: URLSearchParams) => void,
    replace = false,
    state: App.PageState = this.state,
  ): void {
    const next = new globalThis.URL(this.url.href);
    mutate(next.searchParams);
    const request = { replace, state, url: next };
    this.#pending = request;
    // eslint-disable-next-line svelte/no-navigation-without-resolve -- Preserve the current resolved URL, including embedded mode.
    void goto(next, { replace, reset: false, state }).finally(() => {
      if (this.#pending === request) {
        this.#pending = undefined;
      }
    });
  }
}

export { UrlNavigation };
