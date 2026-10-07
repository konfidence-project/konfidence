/* oxlint-disable eslint/max-statements, typescript/prefer-for-of -- Layering is a linear fixed-point relaxation; the indexed pass counter and the accumulation steps read more clearly kept together than split apart. */
import type { Landscape, Stage, VectorPromotionConfig } from "$lib/konfidence-api/types";

// A stage placed on the promotion graph. `column` is the topological depth in
// the promotion DAG (0 = origin/"dev" on the left, higher = closer to "prod" on
// the right); `row` is the vertical slot within that column.
interface StagePlacement {
  stage: Stage;
  landscapeId: string;
  landscapeName: string;
  column: number;
  row: number;
}

// A landscape that has no stages. It still needs to be visible, so it is laid
// out as a labelled placeholder rather than being dropped from the overview.
interface EmptyLandscapePlacement {
  landscapeId: string;
  landscapeName: string;
  column: number;
  row: number;
}

// A promotion relationship between two stages that both exist in the overview.
interface StageEdge {
  id: string;
  sourceStageKey: string;
  targetStageKey: string;
}

interface LandscapeGraph {
  stages: StagePlacement[];
  emptyLandscapes: EmptyLandscapePlacement[];
  edges: StageEdge[];
  columnCount: number;
}

interface StageReferenceLike {
  landscape?: string;
  name: string;
}

// Stages are addressed across the API by (landscape, name); promotions refer to
// them the same way. This key is the single source of truth for matching and
// for SvelteFlow node ids.
const stageKey = (landscapeId: string, stageName: string): string =>
  JSON.stringify([landscapeId, stageName]);

const referenceKey = (reference: StageReferenceLike): string | undefined =>
  reference.landscape === undefined ? undefined : stageKey(reference.landscape, reference.name);

const byName = (first: string, second: string): number =>
  first.localeCompare(second, undefined, { numeric: true, sensitivity: "base" });

const incomingSources = (edges: readonly StageEdge[]): Map<string, string[]> => {
  const incoming = new Map<string, string[]>();
  for (const edge of edges) {
    const sources = incoming.get(edge.targetStageKey) ?? [];
    sources.push(edge.sourceStageKey);
    incoming.set(edge.targetStageKey, sources);
  }
  return incoming;
};

// The deepest source depth + 1, or 0 when a stage has no (non-self) source.
const deepestSourceDepth = (
  key: string,
  sources: readonly string[],
  depths: ReadonlyMap<string, number>,
): number =>
  sources
    // Ignore self-loops so a cyclic edge cannot inflate a node forever.
    .filter((source) => source !== key)
    .reduce((deepest, source) => Math.max(deepest, (depths.get(source) ?? 0) + 1), 0);

// Longest-path layering: a stage sits one column to the right of its deepest
// promotion source. Stages with no incoming promotion stay at column 0 (the
// "dev"/origin edge of the canvas). Cycles are bounded by capping the number of
// relaxation passes at the node count, after which the layering is frozen.
const assignColumns = (
  stageKeys: readonly string[],
  edges: readonly StageEdge[],
): Map<string, number> => {
  const depths = new Map<string, number>(stageKeys.map((key) => [key, 0]));
  const incoming = incomingSources(edges);
  // Each pass can advance a node by at most one level, so `stageKeys.length`
  // passes are enough to reach the fixed point for any acyclic graph.
  for (let pass = 0; pass < stageKeys.length; pass += 1) {
    let changed = false;
    for (const key of stageKeys) {
      const depth = deepestSourceDepth(key, incoming.get(key) ?? [], depths);
      if (depth > (depths.get(key) ?? 0)) {
        depths.set(key, depth);
        changed = true;
      }
    }
    if (!changed) {
      break;
    }
  }
  return depths;
};

const isRenderableEdge = (
  sourceStageKey: string | undefined,
  targetStageKey: string | undefined,
  stageKeys: ReadonlySet<string>,
): boolean =>
  sourceStageKey !== undefined &&
  targetStageKey !== undefined &&
  sourceStageKey !== targetStageKey &&
  stageKeys.has(sourceStageKey) &&
  stageKeys.has(targetStageKey);

const buildEdges = (
  promotionConfigs: readonly VectorPromotionConfig[],
  stageKeys: ReadonlySet<string>,
): StageEdge[] => {
  const seen = new Set<string>();
  const edges: StageEdge[] = [];
  for (const config of promotionConfigs) {
    // A config carries an aggregate source->target that already describes the
    // stage relationship, so the config pair alone avoids duplicate edges per
    // promotion. VectorTemplate origins (no landscape) and dangling references
    // are filtered out by `isRenderableEdge`.
    const sourceStageKey = referenceKey(config.source);
    const targetStageKey = referenceKey(config.target);
    if (
      sourceStageKey !== undefined &&
      targetStageKey !== undefined &&
      isRenderableEdge(sourceStageKey, targetStageKey, stageKeys)
    ) {
      const id = `${sourceStageKey}=>${targetStageKey}`;
      if (!seen.has(id)) {
        seen.add(id);
        edges.push({ id, sourceStageKey, targetStageKey });
      }
    }
  }
  return edges;
};

const placeStages = (
  stageByKey: ReadonlyMap<string, Stage>,
  columns: ReadonlyMap<string, number>,
  landscapeNames: ReadonlyMap<string, string>,
): { stages: StagePlacement[]; maxColumn: number } => {
  const columnBuckets = new Map<number, StagePlacement[]>();
  for (const [key, stage] of stageByKey) {
    const column = columns.get(key) ?? 0;
    const bucket = columnBuckets.get(column) ?? [];
    bucket.push({
      column,
      landscapeId: stage.landscapeId,
      landscapeName:
        landscapeNames.get(stage.landscapeId) ?? `Unknown landscape (${stage.landscapeId})`,
      row: 0,
      stage,
    });
    columnBuckets.set(column, bucket);
  }

  const stages: StagePlacement[] = [];
  let maxColumn = -1;
  const orderedColumns = [...columnBuckets.entries()].toSorted(
    ([first], [second]) => first - second,
  );
  for (const [column, bucket] of orderedColumns) {
    // Stack each column alphabetically by stage name for a stable, readable
    // vertical order.
    const ordered = bucket.toSorted((first, second) => byName(first.stage.name, second.stage.name));
    ordered.forEach((placement, index) => {
      stages.push({ ...placement, row: index });
    });
    maxColumn = Math.max(maxColumn, column);
  }
  return { maxColumn, stages };
};

// Builds the promotion-driven graph: stage nodes positioned dev(left)->prod
// (right) by promotion depth, edges between promoted stages, and labelled
// placeholders for landscapes that have no stages yet.
const buildLandscapeGraph = (
  landscapes: readonly Landscape[],
  stages: readonly Stage[],
  promotionConfigs: readonly VectorPromotionConfig[],
): LandscapeGraph => {
  const landscapeNames = new Map(landscapes.map(({ id, name }) => [id, name]));
  const stageByKey = new Map<string, Stage>(
    stages.map((stage) => [stageKey(stage.landscapeId, stage.name), stage]),
  );
  const stageKeys = new Set(stageByKey.keys());

  const edges = buildEdges(promotionConfigs, stageKeys);
  const columns = assignColumns([...stageKeys], edges);
  const { stages: placedStages, maxColumn } = placeStages(stageByKey, columns, landscapeNames);

  // Landscapes with no stages get a placeholder in a dedicated trailing column
  // so they remain visible and discoverable.
  const stagedLandscapeIds = new Set(stages.map((stage) => stage.landscapeId));
  const emptyColumn = maxColumn + 1;
  const emptyLandscapes: EmptyLandscapePlacement[] = landscapes
    .filter((landscape) => !stagedLandscapeIds.has(landscape.id))
    .map((landscape, index) => ({
      column: emptyColumn,
      landscapeId: landscape.id,
      landscapeName: landscape.name,
      row: index,
    }));

  const columnCount = emptyLandscapes.length > 0 ? emptyColumn + 1 : maxColumn + 1;

  return {
    columnCount: Math.max(columnCount, 0),
    edges,
    emptyLandscapes,
    stages: placedStages,
  };
};

export { buildLandscapeGraph, stageKey };
export type { EmptyLandscapePlacement, LandscapeGraph, StageEdge, StagePlacement };
