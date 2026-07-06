import { type Edge, type EdgeMarker, MarkerType, type Node, Position } from "@xyflow/svelte";

import type { Stage } from "$lib/stages.js";
import type { StageCardVariant } from "$lib/components/stage/variants.js";

type EdgeStatus = "failed" | "running" | "healthy";
type ConditionStatus = "True" | "False" | "Unknown";

interface LandscapeLayout {
  order: string[];
  namespaceIndex: Record<string, number>;
}

const LANDSCAPE_ORDER = ["development", "staging", "production"];
const COLUMN_GAP = 600;
const ROW_GAP = 300;
const EDGE_COLORS: Record<EdgeStatus, string> = {
  failed: "var(--sapNegativeElementColor)",
  healthy: "var(--sapPositiveElementColor)",
  running: "var(--sapInformativeElementColor)",
};
const EDGE_STATUS_BY_READY: Record<ConditionStatus, EdgeStatus> = {
  False: "failed",
  True: "healthy",
  Unknown: "running",
};
const EDGE_LABEL_BY_STATUS: Record<EdgeStatus, string> = {
  failed: "failed",
  healthy: "promote",
  running: "in progress",
};

const readyStatus = (stage: Stage) =>
  stage.status.conditions?.find((condition) => condition.type === "Ready")?.status ?? "Unknown";

const laneKey = (stage: Stage) => stage.metadata.name.split("-").at(-1) ?? stage.metadata.name;

const laneLookupKey = (namespace: string, stage: Stage) => `${namespace}:${laneKey(stage)}`;

/**
 * Known namespaces first (in LANDSCAPE_ORDER), then any custom namespaces alphabetically.
 */
const landscapeOrder = (items: Stage[]) => {
  const namespaces = new Set(items.map((stage) => stage.metadata.namespace));
  const known = LANDSCAPE_ORDER.filter((namespace) => namespaces.has(namespace));
  const custom = [...namespaces]
    .filter((namespace) => !LANDSCAPE_ORDER.includes(namespace))
    .toSorted();

  return [...known, ...custom];
};

const indexByNamespace = (order: string[]): Record<string, number> =>
  Object.fromEntries(order.map((namespace, index) => [namespace, index]));

const sortStages = (items: Stage[], order: string[]) =>
  [...items].toSorted((firstStage, secondStage) => {
    const namespaceDiff =
      order.indexOf(firstStage.metadata.namespace) - order.indexOf(secondStage.metadata.namespace);

    if (namespaceDiff !== 0) {
      return namespaceDiff;
    }
    return firstStage.metadata.name.localeCompare(secondStage.metadata.name);
  });

/**
 * One flow node per stage: namespaces are columns (per `order`),
 * stages within a namespace stack vertically.
 */
const buildNodes = (items: Stage[], layout: LandscapeLayout, variant: StageCardVariant): Node[] => {
  const { order, namespaceIndex } = layout;
  const rowByNamespace: Record<string, number> = {};

  return sortStages(items, order).map((stage) => {
    const row = rowByNamespace[stage.metadata.namespace] ?? 0;
    rowByNamespace[stage.metadata.namespace] = row + 1;

    const position = Object.fromEntries([
      ["x", namespaceIndex[stage.metadata.namespace] * COLUMN_GAP],
      ["y", row * ROW_GAP],
    ]) as Node["position"];

    return {
      data: {
        stage,
        variant,
      },
      id: stage.metadata.name,
      position,
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
      type: "stage",
    };
  });
};

const describeEdge = (stage: Stage) => {
  const ready = readyStatus(stage);
  const status = EDGE_STATUS_BY_READY[ready];
  const label = EDGE_LABEL_BY_STATUS[status];
  const color = EDGE_COLORS[status];

  const markerEnd: EdgeMarker = {
    color,
    height: 18,
    type: MarkerType.ArrowClosed,
    width: 18,
  };

  return { animated: ready === "Unknown", color, label, markerEnd, status };
};

/**
 * Promotion edges linking each stage to its upstream source in the
 * previous namespace (matched by lane key). First-namespace stages
 * have no upstream and produce no edge.
 */
const buildEdges = (items: Stage[], layout: LandscapeLayout): Edge[] => {
  const { order, namespaceIndex } = layout;
  const byNamespaceAndLane: Record<string, Stage> = {};

  for (const stage of items) {
    byNamespaceAndLane[laneLookupKey(stage.metadata.namespace, stage)] = stage;
  }

  return sortStages(items, order).flatMap((stage) => {
    const currentIndex = namespaceIndex[stage.metadata.namespace];
    if (currentIndex <= 0) {
      return [];
    }

    const source = byNamespaceAndLane[laneLookupKey(order[currentIndex - 1], stage)];
    if (!source) {
      return [];
    }

    const { animated, label, markerEnd, status } = describeEdge(stage);

    return {
      animated,
      data: { status },
      id: `${source.metadata.name}-${stage.metadata.name}`,
      label,
      markerEnd,
      source: source.metadata.name,
      target: stage.metadata.name,
      type: "promotion",
    };
  });
};

export { buildEdges, buildNodes, indexByNamespace, landscapeOrder };
export type { EdgeStatus, LandscapeLayout };
