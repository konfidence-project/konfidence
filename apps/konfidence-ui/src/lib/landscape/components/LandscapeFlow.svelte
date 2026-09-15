<script lang="ts">
    /* oxlint-disable eslint/id-length -- SvelteFlow positions use x and y coordinates. */
    import "@xyflow/svelte/dist/style.css";
    import { Controls, SvelteFlow } from "@xyflow/svelte";
    import type { Node } from "@xyflow/svelte";
    import type { Landscape, Stage } from "$lib/landscape/landscapeApi";
    import { groupStages } from "$lib/landscape/stageGrouping";
    import { stageDetailsUrl } from "$lib/projects/url";
    import { themeStore } from "$lib/theme";
    import StageNode from "$lib/landscape/components/StageNode.svelte";

    interface Props {
        landscapes: readonly Landscape[];
        stages: readonly Stage[];
        projectId: string;
        embedded: boolean;
    }
    let { landscapes, stages, projectId, embedded }: Props = $props();

    const nodeTypes = { stage: StageNode };
    // Cards have a fixed height with bounded vector previews; full references are on the details page.
    const CARD_WIDTH = 304;
    const CARD_HEIGHT = 256;
    const GAP = 24;
    const CATEGORY_GAP = 80;
    const categories = ["dev", "test", "prod", "other"] as const;
    const CONTENT_WIDTH =
        categories.length * (CARD_WIDTH + CATEGORY_GAP) - CATEGORY_GAP;
    const MIN_READABLE_ZOOM = 0.75;
    let canvasWidth = $state(0);
    const initialViewport = $derived({
        x: GAP,
        y: GAP,
        zoom: Math.max(
            MIN_READABLE_ZOOM,
            Math.min(1, (canvasWidth - GAP - GAP) / CONTENT_WIDTH),
        ),
    });
    const labels = { dev: "Dev", other: "Other", prod: "Prod", test: "Test" };
    const defaults = {
        connectable: false,
        deletable: false,
        draggable: false,
        focusable: false,
        selectable: false,
    };

    const nodes = $derived.by((): Node[] => {
        const grouped = groupStages(stages);
        const landscapeNames = new Map(
            landscapes.map(({ id, name }) => [id, name]),
        );

        // Each category is a column; API order sets the vertical card positions within it.
        return categories.flatMap((category, column) =>
            // oxlint-disable-next-line oxc/no-map-spread -- Each stage needs an independent node with shared interaction defaults.
            grouped[category].map((stage, row) => ({
                ...defaults,
                data: {
                    category: labels[category],
                    href: stageDetailsUrl({
                        embedded,
                        landscapeId: stage.landscapeId,
                        projectId,
                        stageId: stage.id,
                    }),
                    landscapeName:
                        landscapeNames.get(stage.landscapeId) ??
                        `Unknown landscape (${stage.landscapeId})`,
                    stage,
                },
                height: CARD_HEIGHT,
                id: JSON.stringify(["stage", stage.landscapeId, stage.id]),
                position: {
                    x: column * (CARD_WIDTH + CATEGORY_GAP),
                    y: row * (CARD_HEIGHT + GAP),
                },
                type: "stage",
                width: CARD_WIDTH,
            })),
        );
    });
</script>

<div
    bind:clientWidth={canvasWidth}
    class="h-full min-h-0 min-w-0 bg-[var(--surface-canvas)] [&_.svelte-flow]:[--xy-background-color:var(--surface-canvas)] [&_.svelte-flow]:[--xy-controls-button-background-color:var(--surface-default)] [&_.svelte-flow]:[--xy-controls-button-color:var(--text-primary)] [&_.svelte-flow]:[--xy-controls-button-border-color:var(--border-subtle)]"
    data-testid="landscape-flow"
>
    {#if canvasWidth > 0}
        <SvelteFlow
            {nodes}
            {nodeTypes}
            {initialViewport}
            edges={[]}
            colorMode={themeStore.mode}
            minZoom={0.1}
            maxZoom={1.5}
            nodesDraggable={false}
            nodesConnectable={false}
            nodesFocusable={false}
            elementsSelectable={false}
            selectionOnDrag={false}
            edgesFocusable={false}
            deleteKey={null}
            selectionKey={null}
            multiSelectionKey={null}
        >
            <Controls showLock={false} />
        </SvelteFlow>
    {/if}
</div>
