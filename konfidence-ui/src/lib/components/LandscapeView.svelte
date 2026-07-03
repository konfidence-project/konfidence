<script lang="ts">
    import {
        Controls,
        SvelteFlow,
        type Edge,
        type Node,
        Position,
    } from "@xyflow/svelte";
    import "@xyflow/svelte/dist/style.css";

    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";
    import type { Stage } from "$lib/stages.js";

    const { stages } = $props<{ stages: Stage[] }>();
    const stageCardVariant = getStageCardVariantPreference();
    const nodeTypes = { stage: LandscapeStageNode };

    const version = (stage: Stage) =>
        stage.spec.vector.split(":").at(-1) ?? stage.spec.vector;

    const readyStatus = (stage: Stage) =>
        stage.status.conditions?.find((condition) => condition.type === "Ready")
            ?.status ?? "Unknown";

    const buildNodes = (items: Stage[]): Node[] =>
        items.map((stage, index) => ({
            id: stage.metadata.name,
            type: "stage",
            position: { x: index * 360, y: 0 },
            data: {
                stage,
                variant: stageCardVariant.selected,
            },
            sourcePosition: Position.Right,
            targetPosition: Position.Left,
        }));

    const buildEdges = (items: Stage[]): Edge[] =>
        items.slice(1).map((stage, index) => ({
            id: `${items[index].metadata.name}-${stage.metadata.name}`,
            source: items[index].metadata.name,
            target: stage.metadata.name,
            label: "promote",
            type: "smoothstep",
            animated: readyStatus(stage) === "Unknown",
        }));

    let nodes = $derived(buildNodes(stages));
    let edges = $derived(buildEdges(stages));
</script>

<div class="flow" aria-label="Stage promotion landscape">
    <SvelteFlow
        {nodes}
        {edges}
        {nodeTypes}
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

    :global(.svelte-flow__edge-path) {
        stroke: var(--sapContent_ForegroundBorderColor);
    }

    :global(.svelte-flow__edge.animated .svelte-flow__edge-path) {
        stroke: var(--sapInformativeElementColor);
    }
</style>
