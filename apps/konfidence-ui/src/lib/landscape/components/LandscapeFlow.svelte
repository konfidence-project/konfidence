<script lang="ts">
    /* oxlint-disable eslint/id-length -- SvelteFlow positions use x and y coordinates. */
    import "@xyflow/svelte/dist/style.css";
    import {
        Controls,
        PanOnScrollMode,
        Position,
        SvelteFlow,
    } from "@xyflow/svelte";
    import type { Edge, Node } from "@xyflow/svelte";
    import type {
        Landscape,
        Stage,
        VectorPromotionConfig,
    } from "$lib/konfidence-api/types";
    import { buildLandscapeGraph, stageKey } from "$lib/landscape/landscapeGraph";
    import { stageDetailsUrl } from "$lib/projects/url";
    import { isDarkMode, themeStore } from "$lib/theme";
    import LandscapeCanvasResize from "$lib/landscape/components/LandscapeCanvasResize.svelte";
    import LandscapeEmptyNode from "$lib/landscape/components/LandscapeEmptyNode.svelte";
    import StageNode from "$lib/landscape/components/StageNode.svelte";

    interface Props {
        landscapes: readonly Landscape[];
        stages: readonly Stage[];
        promotionConfigs: readonly VectorPromotionConfig[];
        projectId: string;
        embedded: boolean;
    }
    let {
        landscapes,
        stages,
        promotionConfigs,
        projectId,
        embedded,
    }: Props = $props();

    const nodeTypes = {
        "landscape-empty": LandscapeEmptyNode,
        stage: StageNode,
    };
    // Cards have a fixed height with bounded vector previews; full references are on the details page.
    const CARD_WIDTH = 304;
    const CARD_HEIGHT = 256;
    const GAP = 24;
    // Columns are spaced like the former category columns, leaving room for the
    // dev(left)->prod(right) promotion edges drawn between them.
    const COLUMN_GAP = 80;
    const MIN_READABLE_ZOOM = 0.75;
    let canvasWidth = $state(0);

    const graph = $derived(
        buildLandscapeGraph(landscapes, stages, promotionConfigs),
    );
    const contentWidth = $derived(
        Math.max(1, graph.columnCount) * (CARD_WIDTH + COLUMN_GAP) - COLUMN_GAP,
    );
    const readableZoom = (width: number): number =>
        Math.max(
            MIN_READABLE_ZOOM,
            Math.min(1, (width - GAP - GAP) / contentWidth),
        );
    const defaults = {
        connectable: false,
        deletable: false,
        draggable: false,
        focusable: false,
        selectable: false,
    };

    const columnX = (column: number): number => column * (CARD_WIDTH + COLUMN_GAP);
    const rowY = (row: number): number => row * (CARD_HEIGHT + GAP);

    const nodes = $derived.by((): Node[] => {
        const stageNodes: Node[] = graph.stages.map((placement) => ({
            ...defaults,
            data: {
                href: stageDetailsUrl({
                    embedded,
                    landscapeId: placement.landscapeId,
                    projectId,
                    stageId: placement.stage.id,
                }),
                landscapeName: placement.landscapeName,
                stage: placement.stage,
            },
            height: CARD_HEIGHT,
            id: stageKey(placement.landscapeId, placement.stage.name),
            position: { x: columnX(placement.column), y: rowY(placement.row) },
            sourcePosition: Position.Right,
            targetPosition: Position.Left,
            type: "stage",
            width: CARD_WIDTH,
        }));
        const emptyNodes: Node[] = graph.emptyLandscapes.map((placement) => ({
            ...defaults,
            data: { landscapeName: placement.landscapeName },
            height: CARD_HEIGHT,
            id: JSON.stringify(["landscape-empty", placement.landscapeId]),
            position: { x: columnX(placement.column), y: rowY(placement.row) },
            type: "landscape-empty",
            width: CARD_WIDTH,
        }));
        return [...stageNodes, ...emptyNodes];
    });

    const edges = $derived.by((): Edge[] =>
        graph.edges.map((edge) => ({
            id: edge.id,
            source: edge.sourceStageKey,
            target: edge.targetStageKey,
        })),
    );

    const colorMode = $derived(isDarkMode(themeStore.mode) ? "dark" : "light");
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
            initialViewport={{
                x: GAP,
                y: GAP,
                zoom: readableZoom(canvasWidth),
            }}
            {edges}
            {colorMode}
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
            panOnScroll
            panOnScrollMode={PanOnScrollMode.Free}
        >
            <Controls showLock={false} />
            <LandscapeCanvasResize
                {canvasWidth}
                {contentWidth}
                gap={GAP}
                minReadableZoom={MIN_READABLE_ZOOM}
            />
        </SvelteFlow>
    {/if}
</div>
