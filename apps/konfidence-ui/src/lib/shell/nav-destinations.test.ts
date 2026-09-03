import { describe, expect, it } from "vitest";
import { isActive } from "$lib/shell/nav-destinations";

describe("isActive", () => {
  it("matches an exact path", () => {
    expect(isActive("/projects/p/landscape", "/projects/p/landscape")).toBe(true);
  });

  it("matches a nested path", () => {
    expect(isActive("/projects/p/landscape/detail", "/projects/p/landscape")).toBe(true);
  });

  it("does not match unrelated paths that share a prefix", () => {
    expect(isActive("/projects/p/landscape-x", "/projects/p/landscape")).toBe(false);
  });

  it("does not match a different destination", () => {
    expect(isActive("/projects/p/vector-deployments", "/projects/p/landscape")).toBe(false);
  });
});
