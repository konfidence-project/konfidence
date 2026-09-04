import { describe, expect, it, vi } from "vitest";

import { resolveInitialTheme } from "./resolve.js";
import { DEFAULT_THEME, THEME_QUERY_PARAM, THEME_STORAGE_KEY } from "./constants.js";

const makeLocation = (search: string, pathname = "/dashboard", hash = ""): Location =>
  ({ hash, pathname, search }) as Location;

const makeStorage = (initial: Record<string, string> = {}): Storage => {
  const map = new Map(Object.entries(initial));
  return {
    clear: () => map.clear(),
    getItem: (key: string) => map.get(key) ?? null,
    key: () => null,
    get length() {
      return map.size;
    },
    removeItem: (key: string) => map.delete(key),
    setItem: (key: string, value: string) => {
      map.set(key, value);
    },
  };
};

describe("resolveInitialTheme", () => {
  it("returns the default when no query param and no storage value", () => {
    const theme = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage(),
    });
    expect(theme).toBe(DEFAULT_THEME);
  });

  it("returns the persisted value when present and valid", () => {
    const theme = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage({ [THEME_STORAGE_KEY]: "konfidence-dark" }),
    });
    expect(theme).toBe("konfidence-dark");
  });

  it("ignores invalid persisted values and falls back to default", () => {
    const theme = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage({ [THEME_STORAGE_KEY]: "cosmic" }),
    });
    expect(theme).toBe(DEFAULT_THEME);
  });

  it("prefers a valid query parameter over any persisted value", () => {
    const theme = resolveInitialTheme({
      location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence-dark`),
      storage: makeStorage({ [THEME_STORAGE_KEY]: "konfidence" }),
    });
    expect(theme).toBe("konfidence-dark");
  });

  it("persists the query-parameter value as the new preference", () => {
    const storage = makeStorage();
    resolveInitialTheme({
      location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence-dark`),
      storage,
    });
    expect(storage.getItem(THEME_STORAGE_KEY)).toBe("konfidence-dark");
  });

  it("strips the query parameter from the URL via replaceState", () => {
    const replaceState = vi.fn();
    resolveInitialTheme({
      history: { replaceState },
      location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence-dark&keep=1`, "/x", "#h"),
      storage: makeStorage(),
    });
    expect(replaceState).toHaveBeenCalledWith(null, "", "/x?keep=1#h");
  });

  it("emits a hash-less, query-less URL when no other params remain", () => {
    const replaceState = vi.fn();
    resolveInitialTheme({
      history: { replaceState },
      location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence`, "/dashboard", ""),
      storage: makeStorage(),
    });
    expect(replaceState).toHaveBeenCalledWith(null, "", "/dashboard");
  });

  it("ignores unknown query values and falls through to storage", () => {
    const theme = resolveInitialTheme({
      location: makeLocation(`?${THEME_QUERY_PARAM}=cosmic`),
      storage: makeStorage({ [THEME_STORAGE_KEY]: "konfidence-dark" }),
    });
    expect(theme).toBe("konfidence-dark");
  });

  it("swallows setItem quota errors", () => {
    const storage: Storage = {
      clear: () => undefined,
      getItem: () => null,
      key: () => null,
      length: 0,
      removeItem: () => undefined,
      setItem: () => {
        throw new Error("QuotaExceeded");
      },
    };
    expect(() =>
      resolveInitialTheme({
        location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence-dark`),
        storage,
      }),
    ).not.toThrow();
  });

  it("swallows replaceState errors", () => {
    const replaceState = () => {
      throw new Error("SecurityError");
    };
    expect(() =>
      resolveInitialTheme({
        history: { replaceState },
        location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence-dark`),
        storage: makeStorage(),
      }),
    ).not.toThrow();
  });
});
