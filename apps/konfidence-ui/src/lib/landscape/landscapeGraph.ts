/* oxlint-disable eslint/max-statements, typescript/prefer-for-of -- Layering is a linear fixed-point relaxation; the indexed pass counter and the accumulation steps read more clearly kept together than split apart. */
import type { Landscape, Stage, VectorPromotionConfig } from "$lib/konfidence-api/types";

interface StagePlacement {
  stage: Stage;
  landscapeId: string;
  landscapeName: string;
  column: number;
  row: number;
}

interface EmptyLandscapePlacement {
  landscapeId: string;
  landscapeName: string;
  column: number;
  row: number;
}

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

const deepestSourceDepth = (
  key: string,
  sources: readonly string[],
  depths: ReadonlyMap<string, number>,
): number =>
  sources
    .filter((source) => source !== key)
    .reduce((deepest, source) => Math.max(deepest, (depths.get(source) ?? 0) + 1), 0);

const assignColumns = (
  stageKeys: readonly string[],
  edges: readonly StageEdge[],
): Map<string, number> => {
  const depths = new Map<string, number>(stageKeys.map((key) => [key, 0]));
  const incoming = incomingSources(edges);
  // Bound relaxation by the node count so cyclic graphs cannot loop forever.
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
    const ordered = bucket.toSorted((first, second) => byName(first.stage.name, second.stage.name));
    ordered.forEach((placement, index) => {
      stages.push({ ...placement, row: index });
    });
    maxColumn = Math.max(maxColumn, column);
  }
  return { maxColumn, stages };
};

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

  // Keep empty landscapes discoverable in a dedicated trailing column.
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
