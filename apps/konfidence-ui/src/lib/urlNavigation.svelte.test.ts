import { flushSync } from "svelte";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UrlNavigation } from "./urlNavigation.svelte.ts";

const navigation = vi.hoisted(() => ({
  goto: vi.fn<(url: URL, options: { replace: boolean; state: App.PageState }) => Promise<void>>(),
}));
const appState = vi.hoisted(() => ({
  page: {
    state: {} as App.PageState,
    url: new URL("http://localhost/promotions?embedded=1"),
  },
}));

vi.mock("$app/navigation", () => ({ goto: navigation.goto }));
vi.mock("$app/state", () => appState);

beforeEach(() => {
  navigation.goto.mockReset();
  appState.page.state = {};
  appState.page.url = new URL("http://localhost/promotions?embedded=1");
});

describe("UrlNavigation", () => {
  it("composes pending edits with their history mode and page state", async () => {
    let complete = (): void => {};
    navigation.goto.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve;
        }),
    );
    const url = new UrlNavigation();
    url.update((params) => params.set("promotionId", "a"), false, { overlayOrigin: "promotion" });
    url.update((params) => params.set("promotionConfig", "flow"), url.replace);
    url.update((params) => params.set("promotionId", "b"), url.replace);

    const [next, options] = navigation.goto.mock.calls[2]!;
    expect(next.search).toBe("?embedded=1&promotionId=b&promotionConfig=flow");
    expect(options).toMatchObject({ replace: false, state: { overlayOrigin: "promotion" } });
    complete();
    await Promise.resolve();
  });

  it("keeps the newest pending URL when an older navigation completes", async () => {
    const completions: (() => void)[] = [];
    navigation.goto.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          completions.push(resolve);
        }),
    );
    const url = new UrlNavigation();
    url.update((params) => params.set("deployment", "a"));
    url.update((params) => params.set("deployment", "b"));
    completions[0]!();
    await Promise.resolve();
    expect(url.pending).toBe(true);
    expect(url.url.searchParams.get("deployment")).toBe("b");
    completions[1]!();
    await Promise.resolve();
    expect(url.pending).toBe(false);
  });

  it("reactively clears pending state even when the committed URL does not change", async () => {
    let complete = (): void => {};
    navigation.goto.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve;
        }),
    );
    const url = new UrlNavigation();
    const observed: boolean[] = [];
    const cleanup = $effect.root(() => {
      $effect(() => {
        observed.push(url.pending);
      });
    });
    try {
      flushSync();
      url.update(() => {});
      flushSync();
      complete();
      await Promise.resolve();
      flushSync();
      expect(observed).toEqual([false, true, false]);
    } finally {
      cleanup();
    }
  });
});
