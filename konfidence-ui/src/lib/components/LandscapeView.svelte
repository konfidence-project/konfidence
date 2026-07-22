<script lang="ts">
    import "@xyflow/svelte/dist/style.css";
    import { type EdgeTypes, SvelteFlow } from "@xyflow/svelte";

    import {
        buildEdges,
        buildNodes,
        indexByNamespace,
        landscapeOrder,
    } from "$lib/components/utils/landscape.js";
    import LandscapePromotionEdge from "$lib/components/LandscapePromotionEdge.svelte";
    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import type { Stage } from "$lib/stages.js";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";

    const { stages } = $props<{ stages: Stage[] }>();
    const stageCardVariant = getStageCardVariantPreference();
    const nodeTypes = { stage: LandscapeStageNode };
    const edgeTypes: EdgeTypes = { promotion: LandscapePromotionEdge };

    const order = $derived(landscapeOrder(stages));
    const layout = $derived({
        namespaceIndex: indexByNamespace(order),
        order,
    });
    const nodes = $derived(
        buildNodes(stages, layout, stageCardVariant.selected),
    );
    const edges = $derived(buildEdges(stages, layout));
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
        height: 100%;
        min-width: 0;
        min-height: 0;
        overflow: hidden;
    }

    /* SvelteFlow's root must fill the .flow container explicitly, otherwise
       it collapses to its intrinsic (zero) height inside a flex parent. */
    .flow :global(.svelte-flow) {
        width: 100%;
        height: 100%;
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
