<script lang="ts">
    import "@xyflow/svelte/dist/style.css";
    import { type Node, Position, SvelteFlow } from "@xyflow/svelte";

    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import type { Landscape, Stage } from "$lib/stages";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";
    import { t } from "$lib/stores/i18n.svelte";

    const COLUMN_GAP = 600;
    const ROW_GAP = 300;

    const { landscapes, stages }: { landscapes: Landscape[]; stages: Stage[] } = $props();
    const stageCardVariant = getStageCardVariantPreference();
    const nodeTypes = { stage: LandscapeStageNode };

    const nodes = $derived.by((): Node[] => {
        const landscapeIndex = Object.fromEntries(
            landscapes.map((landscape, index) => [landscape.id, index]),
        );
        const rowByLandscape: Record<string, number> = {};

        return stages.map((stage) => {
            const row = rowByLandscape[stage.landscapeId] ?? 0;
            rowByLandscape[stage.landscapeId] = row + 1;

            return {
                data: { stage, variant: stageCardVariant.selected },
                id: stage.id,
                position: {
                    x: (landscapeIndex[stage.landscapeId] ?? landscapes.length) * COLUMN_GAP,
                    y: row * ROW_GAP,
                },
                sourcePosition: Position.Right,
                targetPosition: Position.Left,
                type: "stage",
            };
        });
    });
</script>

<div class="flow" aria-label={t("LANDSCAPE_ARIA")}>
    <SvelteFlow
        {nodes}
        edges={[]}
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
</style>
