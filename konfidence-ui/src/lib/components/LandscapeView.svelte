<script lang="ts">
    import {
        Controls,
        SvelteFlow,
        type Edge,
        type Node,
        Position,
    } from "@xyflow/svelte";
    import "@xyflow/svelte/dist/style.css";

    import type { Stage } from "$lib/stages.js";

    const { stages } = $props<{ stages: Stage[] }>();

    const version = (stage: Stage) =>
        stage.spec.vector.split(":").at(-1) ?? stage.spec.vector;

    const readyStatus = (stage: Stage) =>
        stage.status.conditions?.find((condition) => condition.type === "Ready")
            ?.status ?? "Unknown";

    const buildNodes = (items: Stage[]): Node[] =>
        items.map((stage, index) => ({
            id: stage.metadata.name,
            position: { x: index * 280, y: 0 },
            data: {
                label: `${stage.metadata.name}\n${version(stage)} · ${readyStatus(stage)}`,
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

    :global(.svelte-flow__node-default) {
        width: 12rem;
        border-color: var(--sapList_BorderColor);
        border-radius: 0.75rem;
        box-shadow: var(--sapContent_Shadow0);
        color: var(--sapTextColor);
        background: var(--sapTile_Background);
        font-family: var(--sapFontFamily), sans-serif;
        white-space: pre-line;
    }

    :global(.svelte-flow__edge-path) {
        stroke: var(--sapContent_ForegroundBorderColor);
    }

    :global(.svelte-flow__edge.animated .svelte-flow__edge-path) {
        stroke: var(--sapInformativeElementColor);
    }
</style>
