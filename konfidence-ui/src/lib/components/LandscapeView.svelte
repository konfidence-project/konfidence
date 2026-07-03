<script lang="ts">
    import {
        MarkerType,
        SvelteFlow,
        type Edge,
        type EdgeMarker,
        type EdgeTypes,
        type Node,
        Position,
    } from "@xyflow/svelte";
    import "@xyflow/svelte/dist/style.css";

    import LandscapePromotionEdge from "$lib/components/LandscapePromotionEdge.svelte";
    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";
    import type { Stage } from "$lib/stages.js";

    const LANDSCAPE_ORDER = ["development", "staging", "production"];
    const COLUMN_GAP = 600;
    const ROW_GAP = 300;

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
            .sort();

        return [...known, ...custom];
    };

    const sortStages = (items: Stage[], order: string[]) =>
        [...items].sort((a, b) => {
            const namespaceDiff =
                order.indexOf(a.metadata.namespace) -
                order.indexOf(b.metadata.namespace);

            if (namespaceDiff !== 0) return namespaceDiff;
            return a.metadata.name.localeCompare(b.metadata.name);
        });

    const buildNodes = (items: Stage[]): Node[] => {
        const order = landscapeOrder(items);
        const rowByNamespace: Record<string, number> = {};

        return sortStages(items, order).map((stage) => {
            const row = rowByNamespace[stage.metadata.namespace] ?? 0;
            rowByNamespace[stage.metadata.namespace] = row + 1;

            return {
                id: stage.metadata.name,
                type: "stage",
                position: {
                    x: order.indexOf(stage.metadata.namespace) * COLUMN_GAP,
                    y: row * ROW_GAP,
                },
                data: {
                    stage,
                    variant: stageCardVariant.selected,
                },
                sourcePosition: Position.Right,
                targetPosition: Position.Left,
            };
        });
    };

    const edgeLabel = (stage: Stage) => {
        const status = readyStatus(stage);

        if (status === "False") return "failed";
        if (status === "Unknown") return "in progress";
        return "promote";
    };

    const edgeStatus = (stage: Stage) => {
        const status = readyStatus(stage);

        if (status === "False") return "failed";
        if (status === "Unknown") return "running";
        return "healthy";
    };

    const edgeColor = (status: ReturnType<typeof edgeStatus>) =>
        status === "failed"
            ? "var(--sapNegativeElementColor)"
            : status === "running"
              ? "var(--sapInformativeElementColor)"
              : "var(--sapPositiveElementColor)";

    const edgeMarker = (status: ReturnType<typeof edgeStatus>): EdgeMarker => ({
        type: MarkerType.ArrowClosed,
        color: edgeColor(status),
        width: 18,
        height: 18,
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
            if (namespaceIndex <= 0) return [];

            const source =
                byNamespaceAndLane[
                    `${order[namespaceIndex - 1]}:${laneKey(stage)}`
                ];
            if (!source) return [];
            const status = edgeStatus(stage);

            return {
                id: `${source.metadata.name}-${stage.metadata.name}`,
                source: source.metadata.name,
                target: stage.metadata.name,
                label: edgeLabel(stage),
                type: "promotion",
                animated: readyStatus(stage) === "Unknown",
                markerEnd: edgeMarker(status),
                data: { status },
            };
        });
    };

    let nodes = $derived(buildNodes(stages));
    let edges = $derived(buildEdges(stages));
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
