import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  PanelDimensions,
  STORAGE_KEY,
  STORAGE_KEYS,
  panelDimensions,
} from "#lib/shell/panelDimensions.svelte.ts";

describe("PanelDimensions", () => {
  beforeEach(() => {
    globalThis.localStorage.removeItem(STORAGE_KEYS.right);
    globalThis.localStorage.removeItem(STORAGE_KEYS.bottom);
  });

  afterEach(() => {
    globalThis.localStorage.removeItem(STORAGE_KEYS.right);
    globalThis.localStorage.removeItem(STORAGE_KEYS.bottom);
    vi.restoreAllMocks();
  });

  it("has no stored preference by default so panels use their own defaults", () => {
    const store = new PanelDimensions();
    expect(store.width).toBeUndefined();
    expect(store.height).toBeUndefined();
  });

  it("persists a width to localStorage and exposes it reactively", () => {
    const store = new PanelDimensions();
    store.setWidth(640);
    expect(store.width).toBe(640);
    expect(globalThis.localStorage.getItem(STORAGE_KEYS.right)).toBe("640");
  });

  it("persists a height to localStorage and exposes it reactively", () => {
    const store = new PanelDimensions();
    store.setHeight(420);
    expect(store.height).toBe(420);
    expect(globalThis.localStorage.getItem(STORAGE_KEYS.bottom)).toBe("420");
  });

  it("keeps width and height independent", () => {
    const store = new PanelDimensions();
    store.setWidth(700);
    store.setHeight(300);
    expect(store.width).toBe(700);
    expect(store.height).toBe(300);
  });

  it("reads previously stored dimensions on construction", () => {
    globalThis.localStorage.setItem(STORAGE_KEYS.right, "720");
    globalThis.localStorage.setItem(STORAGE_KEYS.bottom, "360");
    const store = new PanelDimensions();
    expect(store.width).toBe(720);
    expect(store.height).toBe(360);
  });

  it("preserves the existing width storage key so returning users keep their width", () => {
    expect(STORAGE_KEY).toBe("konfidence:side-panel-width");
    expect(STORAGE_KEYS.right).toBe("konfidence:side-panel-width");
    globalThis.localStorage.setItem("konfidence:side-panel-width", "812");
    expect(new PanelDimensions().width).toBe(812);
  });

  it("ignores a non-numeric or non-positive stored value", () => {
    globalThis.localStorage.setItem(STORAGE_KEYS.right, "not-a-number");
    expect(new PanelDimensions().width).toBeUndefined();
    globalThis.localStorage.setItem(STORAGE_KEYS.bottom, "-10");
    expect(new PanelDimensions().height).toBeUndefined();
  });

  it("rounds fractional dimensions before storing", () => {
    const store = new PanelDimensions();
    store.setWidth(543.6);
    store.setHeight(299.4);
    expect(globalThis.localStorage.getItem(STORAGE_KEYS.right)).toBe("544");
    expect(globalThis.localStorage.getItem(STORAGE_KEYS.bottom)).toBe("299");
  });

  it("returns undefined when localStorage is unavailable or throws", () => {
    vi.spyOn(globalThis.localStorage, "getItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(new PanelDimensions().width).toBeUndefined();
    expect(new PanelDimensions().height).toBeUndefined();
  });

  it("swallows write errors so a throwing localStorage does not crash", () => {
    const store = new PanelDimensions();
    vi.spyOn(globalThis.localStorage, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(() => store.setWidth(500)).not.toThrow();
    expect(() => store.setHeight(300)).not.toThrow();
    expect(store.width).toBe(500);
    expect(store.height).toBe(300);
  });

  it("exports a shared singleton used by every panel consumer", () => {
    expect(panelDimensions).toBeInstanceOf(PanelDimensions);
  });
});
