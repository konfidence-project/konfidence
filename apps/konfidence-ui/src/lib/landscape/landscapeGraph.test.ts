import { describe, expect, it } from "vitest";
import type { components } from "@konfidence/api-client/schema";
import { buildLandscapeGraph, stageKey } from "$lib/landscape/landscapeGraph";

type Landscape = components["schemas"]["Landscape"];
type Stage = components["schemas"]["Stage"];
type VectorPromotion = components["schemas"]["VectorPromotion"];
type VectorPromotionConfig = components["schemas"]["VectorPromotionConfig"];

const landscape = (id: string, name: string): Landscape => ({ id, name });

const stage = (landscapeId: string, name: string): Stage => ({
  id: `${landscapeId}-${name}`,
  landscapeId,
  name,
});

const config = (
  id: string,
  source: { landscape?: string; name: string },
  target: { landscape: string; name: string; promotions?: VectorPromotion[] },
): VectorPromotionConfig => {
  const { promotions = [], ...targetRef } = target;
  return {
    id,
    promotions,
    source: { kind: source.landscape === undefined ? "VectorTemplate" : "Stage", ...source },
    target: { kind: "Stage", ...targetRef },
  };
};

const promotion = (
  sequence: number,
  status: VectorPromotion["status"],
): VectorPromotion => ({
  id: `promotion-${sequence}`,
  sequence,
  source: { kind: "Stage", landscape: "development", name: "dev-a" },
  status,
  target: { kind: "Stage", landscape: "test", name: "test-a" },
  vector: "registry//component:1.0.0",
});

describe("buildLandscapeGraph", () => {
  it("orders promoted stages left-to-right by promotion depth", () => {
    const graph = buildLandscapeGraph(
      [
        landscape("development", "Development"),
        landscape("test", "Test"),
        landscape("production", "Production"),
      ],
      [stage("development", "dev-a"), stage("test", "test-a"), stage("production", "prod-a")],
      [
        config(
          "dev-to-test",
          { landscape: "development", name: "dev-a" },
          { landscape: "test", name: "test-a" },
        ),
        config(
          "test-to-prod",
          { landscape: "test", name: "test-a" },
          { landscape: "production", name: "prod-a" },
        ),
      ],
    );
    const columnOf = (landscapeId: string, name: string): number =>
      graph.stages.find((placement) => placement.stage.id === `${landscapeId}-${name}`)!.column;
    expect(columnOf("development", "dev-a")).toBe(0);
    expect(columnOf("test", "test-a")).toBe(1);
    expect(columnOf("production", "prod-a")).toBe(2);
    expect(graph.edges).toEqual([
      {
        id: `${stageKey("development", "dev-a")}=>${stageKey("test", "test-a")}`,
        sourceStageKey: stageKey("development", "dev-a"),
        targetStageKey: stageKey("test", "test-a"),
      },
      {
        id: `${stageKey("test", "test-a")}=>${stageKey("production", "prod-a")}`,
        sourceStageKey: stageKey("test", "test-a"),
        targetStageKey: stageKey("production", "prod-a"),
      },
    ]);
  });

  it("keeps unpromoted stages in the origin column and stacks them alphabetically", () => {
    const graph = buildLandscapeGraph(
      [landscape("development", "Development")],
      [stage("development", "dev-b"), stage("development", "dev-a"), stage("development", "dev-c")],
      [],
    );
    expect(
      graph.stages.map((placement) => [placement.stage.name, placement.column, placement.row]),
    ).toEqual([
      ["dev-a", 0, 0],
      ["dev-b", 0, 1],
      ["dev-c", 0, 2],
    ]);
  });

  it("uses the deepest source when a stage has multiple incoming promotions", () => {
    const graph = buildLandscapeGraph(
      [
        landscape("development", "Development"),
        landscape("test", "Test"),
        landscape("production", "Production"),
      ],
      [stage("development", "dev-a"), stage("test", "test-a"), stage("production", "prod-a")],
      [
        config(
          "dev-to-prod",
          { landscape: "development", name: "dev-a" },
          { landscape: "production", name: "prod-a" },
        ),
        config(
          "dev-to-test",
          { landscape: "development", name: "dev-a" },
          { landscape: "test", name: "test-a" },
        ),
        config(
          "test-to-prod",
          { landscape: "test", name: "test-a" },
          { landscape: "production", name: "prod-a" },
        ),
      ],
    );
    const columnOf = (name: string): number =>
      graph.stages.find((placement) => placement.stage.name === name)!.column;
    // The prod-a stage is reachable directly (depth 1) and via test-a (depth
    // 2); the deepest path wins so it sits to the right of test-a.
    expect(columnOf("test-a")).toBe(1);
    expect(columnOf("prod-a")).toBe(2);
  });

  it("adds a placeholder for landscapes without stages", () => {
    const graph = buildLandscapeGraph(
      [landscape("development", "Development"), landscape("staging", "Staging")],
      [stage("development", "dev-a")],
      [],
    );
    expect(graph.stages).toHaveLength(1);
    expect(graph.emptyLandscapes).toEqual([
      { column: 1, landscapeId: "staging", landscapeName: "Staging", row: 0 },
    ]);
    expect(graph.columnCount).toBe(2);
  });

  it("ignores VectorTemplate sources and dangling references without dropping stages", () => {
    const graph = buildLandscapeGraph(
      [landscape("development", "Development")],
      [stage("development", "dev-a")],
      [
        config(
          "template-origin",
          { name: "some-template" },
          { landscape: "development", name: "dev-a" },
        ),
        config(
          "dangling",
          { landscape: "development", name: "dev-a" },
          { landscape: "test", name: "missing" },
        ),
      ],
    );
    expect(graph.edges).toHaveLength(0);
    expect(graph.stages).toHaveLength(1);
    expect(graph.stages[0]!.column).toBe(0);
  });

  it("falls back gracefully when promotions form a cycle", () => {
    const graph = buildLandscapeGraph(
      [landscape("a", "A"), landscape("b", "B")],
      [stage("a", "x"), stage("b", "y")],
      [
        config("x-to-y", { landscape: "a", name: "x" }, { landscape: "b", name: "y" }),
        config("y-to-x", { landscape: "b", name: "y" }, { landscape: "a", name: "x" }),
      ],
    );
    // A cycle cannot produce a valid layering; the important contract is that
    // it terminates and keeps both stages on the canvas.
    expect(graph.stages).toHaveLength(2);
    expect(graph.edges).toHaveLength(2);
  });

  it("labels stages whose landscape is unknown instead of dropping them", () => {
    const graph = buildLandscapeGraph([], [stage("ghost", "orphan")], []);
    expect(graph.stages[0]!.landscapeName).toBe("Unknown landscape (ghost)");
  });

  it("surfaces the newest promotion status on the edge regardless of lifecycle phase", () => {
    const graph = buildLandscapeGraph(
      [landscape("development", "Development"), landscape("test", "Test")],
      [stage("development", "dev-a"), stage("test", "test-a")],
      [
        config(
          "dev-to-test",
          { landscape: "development", name: "dev-a" },
          {
            landscape: "test",
            name: "test-a",
            promotions: [promotion(1, "Succeeded"), promotion(3, "Waiting"), promotion(2, "Succeeded")],
          },
        ),
      ],
    );
    // Sequence 3 (Waiting) is the newest promotion, so it wins over the earlier
    // Succeeded runs even though it is not terminal.
    expect(graph.edges).toHaveLength(1);
    expect(graph.edges[0]!.status).toBe("Waiting");
  });

  it("leaves the edge status unset when the config has no promotions", () => {
    const graph = buildLandscapeGraph(
      [landscape("development", "Development"), landscape("test", "Test")],
      [stage("development", "dev-a"), stage("test", "test-a")],
      [
        config(
          "dev-to-test",
          { landscape: "development", name: "dev-a" },
          { landscape: "test", name: "test-a", promotions: [] },
        ),
      ],
    );
    expect(graph.edges[0]!.status).toBeUndefined();
  });

  it("picks the highest-sequence status when configs collapse onto one edge", () => {
    const graph = buildLandscapeGraph(
      [landscape("development", "Development"), landscape("test", "Test")],
      [stage("development", "dev-a"), stage("test", "test-a")],
      [
        config(
          "older",
          { landscape: "development", name: "dev-a" },
          { landscape: "test", name: "test-a", promotions: [promotion(5, "Failed")] },
        ),
        config(
          "newer",
          { landscape: "development", name: "dev-a" },
          { landscape: "test", name: "test-a", promotions: [promotion(9, "Succeeded")] },
        ),
      ],
    );
    // Both configs describe the same stage pair; the config with the highest
    // settled sequence (9, Succeeded) decides the edge colour.
    expect(graph.edges).toHaveLength(1);
    expect(graph.edges[0]!.status).toBe("Succeeded");
  });
});
