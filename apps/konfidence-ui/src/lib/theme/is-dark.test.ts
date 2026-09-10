import { afterEach, describe, expect, it, vi } from "vitest";

import { isDarkMode } from "./is-dark.js";

describe("isDarkMode", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([
    ["light", false],
    ["dark", true],
  ] as const)("resolves %s mode to %s", (mode, expected) => {
    expect(isDarkMode(mode, false)).toBe(expected);
  });

  it("uses the operating system preference for system mode", () => {
    expect(isDarkMode("system", true)).toBe(true);
    expect(isDarkMode("system", false)).toBe(false);
  });

  it("reads the operating system preference when it is not provided", () => {
    const matchMedia = vi.fn().mockReturnValue({ matches: true } as MediaQueryList);
    vi.stubGlobal("matchMedia", matchMedia);

    expect(isDarkMode("system")).toBe(true);

    expect(matchMedia).toHaveBeenCalledWith("(prefers-color-scheme: dark)");
  });
});
