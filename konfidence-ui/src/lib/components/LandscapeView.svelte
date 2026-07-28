<script lang="ts">
    import { type Node, Position, SvelteFlow } from "@xyflow/svelte";

    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import type { Landscape, Stage } from "$lib/stages";
    import { getThemePreference } from "$lib/stores/theme.svelte";

    const COLUMN_GAP = 600;
    const ROW_GAP = 300;

    const { landscapes, stages }: { landscapes: Landscape[]; stages: Stage[] } = $props();
    const theme = getThemePreference();
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
                data: { stage },
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

<div
    class="landscape-flow h-full min-h-0 min-w-0 flex-1 overflow-hidden max-[48rem]:min-h-[calc(100dvh-3.25rem)]"
    aria-label="Stage landscape"
    data-testid="landscape-view"
>
    <h1 class="sr-only">Delivery landscape</h1>
    <SvelteFlow
        {nodes}
        edges={[]}
        {nodeTypes}
        fitView
        colorMode={theme.selected === "konfidence-dark" ? "dark" : "light"}
        nodesDraggable={false}
        nodesConnectable={false}
    ></SvelteFlow>
</div>
