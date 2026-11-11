import { describe, expect, it } from "vitest";
import type { Landscape } from "#lib/konfidence-api/types.ts";
import { landscapeLabel, referenceLabel } from "./promotions.ts";
import type { PromotionReference } from "./promotions.ts";

const landscapes: readonly Landscape[] = [
  { id: "development", name: "Development" },
  { id: "production-us", name: "Production" },
  { id: "production-eu", name: "Production" },
];

describe("promotion reference labels", () => {
  it.each<{ expected: string; id: string | undefined }>([
    { expected: "Development", id: "development" },
    { expected: "Production (production-us)", id: "production-us" },
    { expected: "Production (production-eu)", id: "production-eu" },
    { expected: "retired-landscape", id: "retired-landscape" },
    { expected: "Unresolved landscape", id: undefined },
  ])("labels landscape $id as $expected", ({ expected, id }) => {
    expect(landscapeLabel(id, landscapes)).toBe(expected);
  });

  it.each<{ expected: string; reference: PromotionReference }>([
    {
      expected: "Development/dev",
      reference: { kind: "Stage", landscape: "development", name: "dev" },
    },
    {
      expected: "Production (production-us)/prod",
      reference: { kind: "Stage", landscape: "production-us", name: "prod" },
    },
    {
      expected: "Production (production-eu)/prod",
      reference: { kind: "Stage", landscape: "production-eu", name: "prod" },
    },
    {
      expected: "retired-landscape/prod",
      reference: { kind: "Stage", landscape: "retired-landscape", name: "prod" },
    },
    {
      expected: "Unresolved landscape/dev",
      reference: { kind: "Stage", name: "dev" },
    },
    {
      expected: "VectorTemplate: release-template",
      reference: { kind: "VectorTemplate", name: "release-template" },
    },
  ])("qualifies references as $expected", ({ expected, reference }) => {
    expect(referenceLabel(reference, landscapes)).toBe(expected);
  });

  it("does not invent landscape context for a VectorTemplate", () => {
    expect(
      referenceLabel(
        { kind: "VectorTemplate", landscape: "production-us", name: "release-template" },
        landscapes,
      ),
    ).toBe("VectorTemplate: release-template");
  });
});
