<script lang="ts">
    import "@xyflow/svelte/dist/style.css";
    import { MarkerType, Position, SvelteFlow } from "@xyflow/svelte";

    import LandscapePromotionEdge from "$lib/components/LandscapePromotionEdge.svelte";
    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import type { Stage } from "$lib/stages.js";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";

    type Edge = import("@xyflow/svelte").Edge;
    type EdgeMarker = import("@xyflow/svelte").EdgeMarker;
    type EdgeTypes = import("@xyflow/svelte").EdgeTypes;
    type Node = import("@xyflow/svelte").Node;

    const LANDSCAPE_ORDER = ["development", "staging", "production"];
    const COLUMN_GAP = 600;
    const ROW_GAP = 300;
    const EDGE_COLORS: Record<ReturnType<typeof edgeStatus>, string> = {
        failed: "var(--sapNegativeElementColor)",
        healthy: "var(--sapPositiveElementColor)",
        running: "var(--sapInformativeElementColor)",
    };

    const { stages } = $props<{ stages: Stage[] }>();
    const stageCardVariant = getStageCardVariantPreference();
    const nodeTypes = { stage: LandscapeStageNode };
    const edgeTypes: EdgeTypes = { promotion: LandscapePromotionEdge };

    const readyStatus = (stage: Stage) =>
        stage.status.conditions?.find((condition) => condition.type === "Ready")
            ?.status ?? "Unknown";

    const laneKey = (stage: Stage) =>
        stage.metadata.name.split("-").at(-1) ?? stage.metadata.name;

    const landscapeOrder = (items: Stage[]) => {
        const namespaces = new Set(
            items.map((stage) => stage.metadata.namespace),
        );
        const known = LANDSCAPE_ORDER.filter((namespace) =>
            namespaces.has(namespace),
        );
        const custom = [...namespaces]
            .filter((namespace) => !LANDSCAPE_ORDER.includes(namespace))
            .toSorted();

        return [...known, ...custom];
    };

    const sortStages = (items: Stage[], order: string[]) =>
        [...items].toSorted((firstStage, secondStage) => {
            const namespaceDiff =
                order.indexOf(firstStage.metadata.namespace) -
                order.indexOf(secondStage.metadata.namespace);

            if (namespaceDiff !== 0) {return namespaceDiff;}
            return firstStage.metadata.name.localeCompare(secondStage.metadata.name);
        });

    const buildNodes = (items: Stage[]): Node[] => {
        const order = landscapeOrder(items);
        const rowByNamespace: Record<string, number> = {};

        return sortStages(items, order).map((stage) => {
            const row = rowByNamespace[stage.metadata.namespace] ?? 0;
            rowByNamespace[stage.metadata.namespace] = row + 1;

            const position = Object.fromEntries([
                ["x", order.indexOf(stage.metadata.namespace) * COLUMN_GAP],
                ["y", row * ROW_GAP],
            ]) as Node["position"];

            return {
                data: {
                    stage,
                    variant: stageCardVariant.selected,
                },
                id: stage.metadata.name,
                position,
                sourcePosition: Position.Right,
                targetPosition: Position.Left,
                type: "stage",
            };
        });
    };

    const edgeLabel = (stage: Stage) => {
        const status = readyStatus(stage);

        if (status === "False") {return "failed";}
        if (status === "Unknown") {return "in progress";}
        return "promote";
    };

    const edgeStatus = (stage: Stage) => {
        const status = readyStatus(stage);

        if (status === "False") {return "failed";}
        if (status === "Unknown") {return "running";}
        return "healthy";
    };

    const edgeColor = (status: ReturnType<typeof edgeStatus>) => EDGE_COLORS[status];

    const edgeMarker = (status: ReturnType<typeof edgeStatus>): EdgeMarker => ({
        color: edgeColor(status),
        height: 18,
        type: MarkerType.ArrowClosed,
        width: 18,
    });

    const buildEdges = (items: Stage[]): Edge[] => {
        const order = landscapeOrder(items);
        const byNamespaceAndLane: Record<string, Stage> = {};

        for (const stage of items) {
            byNamespaceAndLane[
                `${stage.metadata.namespace}:${laneKey(stage)}`
            ] = stage;
        }

        return sortStages(items, order).flatMap((stage) => {
            const namespaceIndex = order.indexOf(stage.metadata.namespace);
            if (namespaceIndex <= 0) {return [];}

            const source =
                byNamespaceAndLane[
                    `${order[namespaceIndex - 1]}:${laneKey(stage)}`
                ];
            if (!source) {return [];}
            const status = edgeStatus(stage);

            return {
                animated: readyStatus(stage) === "Unknown",
                data: { status },
                id: `${source.metadata.name}-${stage.metadata.name}`,
                label: edgeLabel(stage),
                markerEnd: edgeMarker(status),
                source: source.metadata.name,
                target: stage.metadata.name,
                type: "promotion",
            };
        });
    };

    const nodes = $derived(buildNodes(stages));
    const edges = $derived(buildEdges(stages));
</script>

<div class="flow" aria-label="Stage promotion landscape">
    <SvelteFlow
        {nodes}
        {edges}
        {nodeTypes}
        {edgeTypes}
        fitView
        colorMode="system"
        nodesDraggable={false}
        nodesConnectable={false}
    ></SvelteFlow>
</div>

<style>
    .flow {
        flex: 1;
        width: 100%;
        overflow: hidden;
    }

    :global(.svelte-flow__node-stage) {
        width: 20rem;
        border: 0;
        background: transparent;
        box-shadow: none;
        color: var(--sapTextColor);
        font-family: var(--sapFontFamily), sans-serif;
        padding: 0;
    }

    :global(
        .svelte-flow__edge:not(.svelte-flow__edge-promotion)
            .svelte-flow__edge-path
    ) {
        stroke: var(--sapContent_ForegroundBorderColor);
    }

    :global(
        .svelte-flow__edge:not(.svelte-flow__edge-promotion).animated
            .svelte-flow__edge-path
    ) {
        stroke: var(--sapInformativeElementColor);
    }
</style>
