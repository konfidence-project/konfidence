import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  PanelDimensions,
  STORAGE_KEY,
  panelDimensions,
} from "#lib/shell/panelDimensions.svelte.ts";

describe("PanelDimensions", () => {
  beforeEach(() => {
    globalThis.localStorage.removeItem(STORAGE_KEY);
  });

  afterEach(() => {
    globalThis.localStorage.removeItem(STORAGE_KEY);
    vi.restoreAllMocks();
  });

  it("has no stored preference by default so panels use their own defaults", () => {
    const store = new PanelDimensions();
    expect(store.width).toBeUndefined();
  });

  it("persists a width to localStorage and exposes it reactively", () => {
    const store = new PanelDimensions();
    store.setWidth(640);
    expect(store.width).toBe(640);
    expect(globalThis.localStorage.getItem(STORAGE_KEY)).toBe("640");
  });

  it("reads the previously stored width on construction", () => {
    globalThis.localStorage.setItem(STORAGE_KEY, "720");
    const store = new PanelDimensions();
    expect(store.width).toBe(720);
  });

  it("preserves the existing width storage key so returning users keep their width", () => {
    expect(STORAGE_KEY).toBe("konfidence:side-panel-width");
    globalThis.localStorage.setItem("konfidence:side-panel-width", "812");
    expect(new PanelDimensions().width).toBe(812);
  });

  it("ignores a non-numeric or non-positive stored value", () => {
    globalThis.localStorage.setItem(STORAGE_KEY, "not-a-number");
    expect(new PanelDimensions().width).toBeUndefined();
    globalThis.localStorage.setItem(STORAGE_KEY, "-10");
    expect(new PanelDimensions().width).toBeUndefined();
  });

  it("rounds fractional widths before storing", () => {
    const store = new PanelDimensions();
    store.setWidth(543.6);
    expect(globalThis.localStorage.getItem(STORAGE_KEY)).toBe("544");
  });

  it("returns undefined when localStorage is unavailable or throws", () => {
    vi.spyOn(globalThis.localStorage, "getItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(new PanelDimensions().width).toBeUndefined();
  });

  it("swallows write errors so a throwing localStorage does not crash", () => {
    const store = new PanelDimensions();
    vi.spyOn(globalThis.localStorage, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(() => store.setWidth(500)).not.toThrow();
    expect(store.width).toBe(500);
  });

  it("exports a shared singleton used by every panel consumer", () => {
    expect(panelDimensions).toBeInstanceOf(PanelDimensions);
  });
});
