<script lang="ts">
    import "@xyflow/svelte/dist/style.css";
    import { type Node, Position, SvelteFlow } from "@xyflow/svelte";

    import LandscapeStageNode from "$lib/components/LandscapeStageNode.svelte";
    import StageCard from "$lib/components/stage/cards/StageCard.svelte";
    import { getThemePreference } from "$lib/theme-preference.svelte";
    import type { Landscape, Stage } from "$lib/stages";

    const COLUMN_GAP = 600;
    const ROW_GAP = 300;

    const { landscapes, stages }: { landscapes: Landscape[]; stages: Stage[] } = $props();
    const theme = getThemePreference();
    const nodeTypes = { stage: LandscapeStageNode };
    const flowColorMode = $derived.by(() => {
        if (theme.selected === "konfidence-dark") {
            return "dark";
        }
        return "light";
    });
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

<section class="flex min-h-0 min-w-0 flex-1" aria-labelledby="landscape-title">
    <h1 id="landscape-title" class="sr-only">Delivery landscape</h1>
    <div class="hidden h-full min-h-0 min-w-0 flex-1 overflow-hidden  min-[52rem]:block" aria-label="Stage landscape">
        <SvelteFlow
            {nodes}
            edges={[]}
            {nodeTypes}
            fitView
            colorMode={flowColorMode}
            nodesDraggable={false}
            nodesConnectable={false}
            minZoom={0.3}
            maxZoom={1.5}
        ></SvelteFlow>
    </div>
    <div class="grid h-full flex-1 gap-3 overflow-y-auto p-4 min-[52rem]:hidden">
        <p class="m-0 text-[0.78rem] font-bold tracking-[0.08em] text-app-muted uppercase">Delivery stages</p>
        <ul class="grid list-none content-start gap-[0.8rem] p-0">
            {#each stages as stage (stage.id)}
                <li><StageCard {stage} /></li>
            {/each}
        </ul>
    </div>
</section>
