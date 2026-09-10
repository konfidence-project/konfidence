import { describe, expect, it } from "vitest";
import { EMBEDDED_ON, EMBEDDED_QUERY, isEmbedded } from "$lib/shell/embedded";

const urlOf = (href: string): URL => new globalThis.URL(href, "http://localhost");

describe("isEmbedded", () => {
  it("returns false when the query parameter is absent", () => {
    expect(isEmbedded(urlOf("/projects/p/landscape"))).toBe(false);
  });

  it("returns true when ?embedded=1", () => {
    expect(isEmbedded(urlOf(`/projects/p/landscape?${EMBEDDED_QUERY}=${EMBEDDED_ON}`))).toBe(true);
  });

  it("returns false for other values", () => {
    expect(isEmbedded(urlOf(`/projects/p/landscape?${EMBEDDED_QUERY}=0`))).toBe(false);
    expect(isEmbedded(urlOf(`/projects/p/landscape?${EMBEDDED_QUERY}=yes`))).toBe(false);
  });
});
