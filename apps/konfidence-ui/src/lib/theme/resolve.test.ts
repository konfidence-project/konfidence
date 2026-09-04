import { describe, expect, it, vi } from "vitest";

import { resolveInitialTheme } from "./resolve.js";
import {
  DEFAULT_MODE,
  DEFAULT_THEME,
  MODE_QUERY_PARAM,
  MODE_STORAGE_KEY,
  THEME_QUERY_PARAM,
  THEME_STORAGE_KEY,
} from "./constants.js";

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
  it("returns the default theme + mode when no query params and no storage", () => {
    const state = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage(),
    });
    expect(state.theme).toBe(DEFAULT_THEME);
    expect(state.mode).toBe(DEFAULT_MODE);
  });

  it("returns the persisted mode when present and valid", () => {
    const state = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage({ [MODE_STORAGE_KEY]: "dark" }),
    });
    expect(state.mode).toBe("dark");
  });

  it("returns the persisted theme when present and valid", () => {
    const state = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage({ [THEME_STORAGE_KEY]: "konfidence" }),
    });
    expect(state.theme).toBe("konfidence");
  });

  it("ignores invalid persisted mode values and falls back to default", () => {
    const state = resolveInitialTheme({
      location: makeLocation(""),
      storage: makeStorage({ [MODE_STORAGE_KEY]: "cosmic" }),
    });
    expect(state.mode).toBe(DEFAULT_MODE);
  });

  it("prefers a valid mode query parameter over any persisted value", () => {
    const state = resolveInitialTheme({
      location: makeLocation(`?${MODE_QUERY_PARAM}=dark`),
      storage: makeStorage({ [MODE_STORAGE_KEY]: "light" }),
    });
    expect(state.mode).toBe("dark");
  });

  it("persists the mode query parameter as the new preference", () => {
    const storage = makeStorage();
    resolveInitialTheme({
      location: makeLocation(`?${MODE_QUERY_PARAM}=dark`),
      storage,
    });
    expect(storage.getItem(MODE_STORAGE_KEY)).toBe("dark");
  });

  it("persists the theme query parameter as the new preference", () => {
    const storage = makeStorage();
    resolveInitialTheme({
      location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence`),
      storage,
    });
    expect(storage.getItem(THEME_STORAGE_KEY)).toBe("konfidence");
  });

  it("supports resolving both axes from a single URL", () => {
    const state = resolveInitialTheme({
      location: makeLocation(`?${THEME_QUERY_PARAM}=konfidence&${MODE_QUERY_PARAM}=dark`),
      storage: makeStorage(),
    });
    expect(state.theme).toBe("konfidence");
    expect(state.mode).toBe("dark");
  });

  it("accepts the `system` mode", () => {
    const state = resolveInitialTheme({
      location: makeLocation(`?${MODE_QUERY_PARAM}=system`),
      storage: makeStorage(),
    });
    expect(state.mode).toBe("system");
  });

  it("strips both query parameters from the URL via replaceState", () => {
    const replaceState = vi.fn();
    resolveInitialTheme({
      history: { replaceState },
      location: makeLocation(
        `?${THEME_QUERY_PARAM}=konfidence&${MODE_QUERY_PARAM}=dark&keep=1`,
        "/x",
        "#h",
      ),
      storage: makeStorage(),
    });
    expect(replaceState).toHaveBeenCalledWith(null, "", "/x?keep=1#h");
  });

  it("emits a hash-less, query-less URL when no other params remain", () => {
    const replaceState = vi.fn();
    resolveInitialTheme({
      history: { replaceState },
      location: makeLocation(`?${MODE_QUERY_PARAM}=dark`, "/dashboard", ""),
      storage: makeStorage(),
    });
    expect(replaceState).toHaveBeenCalledWith(null, "", "/dashboard");
  });

  it("ignores unknown mode query values and falls through to storage", () => {
    const state = resolveInitialTheme({
      location: makeLocation(`?${MODE_QUERY_PARAM}=cosmic`),
      storage: makeStorage({ [MODE_STORAGE_KEY]: "dark" }),
    });
    expect(state.mode).toBe("dark");
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
        location: makeLocation(`?${MODE_QUERY_PARAM}=dark`),
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
        location: makeLocation(`?${MODE_QUERY_PARAM}=dark`),
        storage: makeStorage(),
      }),
    ).not.toThrow();
  });

  it("does not touch replaceState when neither param is present", () => {
    const replaceState = vi.fn();
    resolveInitialTheme({
      history: { replaceState },
      location: makeLocation(""),
      storage: makeStorage(),
    });
    expect(replaceState).not.toHaveBeenCalled();
  });
});
