import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { STORAGE_KEY, SidePanelWidth, sidePanelWidth } from "$lib/shell/sidePanelWidth.svelte";

describe("SidePanelWidth", () => {
  beforeEach(() => {
    globalThis.localStorage.removeItem(STORAGE_KEY);
  });

  afterEach(() => {
    globalThis.localStorage.removeItem(STORAGE_KEY);
  });

  it("has no stored preference by default so the panel uses its own default", () => {
    const store = new SidePanelWidth();
    expect(store.current).toBeUndefined();
  });

  it("persists a width to localStorage and exposes it reactively", () => {
    const store = new SidePanelWidth();
    store.set(640);
    expect(store.current).toBe(640);
    expect(globalThis.localStorage.getItem(STORAGE_KEY)).toBe("640");
  });

  it("reads a previously stored width on construction", () => {
    globalThis.localStorage.setItem(STORAGE_KEY, "720");
    const store = new SidePanelWidth();
    expect(store.current).toBe(720);
  });

  it("ignores a non-numeric or non-positive stored value", () => {
    globalThis.localStorage.setItem(STORAGE_KEY, "not-a-number");
    expect(new SidePanelWidth().current).toBeUndefined();
    globalThis.localStorage.setItem(STORAGE_KEY, "-10");
    expect(new SidePanelWidth().current).toBeUndefined();
  });

  it("rounds fractional widths before storing", () => {
    const store = new SidePanelWidth();
    store.set(543.6);
    expect(globalThis.localStorage.getItem(STORAGE_KEY)).toBe("544");
  });

  it("exports a shared singleton used by every SidePanel consumer", () => {
    expect(sidePanelWidth).toBeInstanceOf(SidePanelWidth);
  });
});
