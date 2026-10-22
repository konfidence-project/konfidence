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
    import type { Landscape, Stage, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
    import { buildLandscapeGraph, stageKey } from "#lib/landscape/landscapeGraph.ts";
    import { mode } from "mode-watcher";
    import LandscapeCanvasResize from "#lib/landscape/components/LandscapeCanvasResize.svelte";
    import LandscapeEmptyNode from "#lib/landscape/components/LandscapeEmptyNode.svelte";
    import StageNode from "#lib/landscape/components/StageNode.svelte";
    import type { StageOverlayReference } from "#lib/landscape/stageOverlay.svelte.ts";

    interface Props {
        landscapes: readonly Landscape[];
        stages: readonly Stage[];
        promotionConfigs: readonly VectorPromotionConfig[];
        selectedStage?: Stage;
        stageHref: (stage: Stage) => string;
        stageReference: (stage: Stage) => StageOverlayReference;
        selectedPromotionId?: string;
        highlightedStageKeys?: readonly string[];
        onPromotionSelect?: (promotionId: string, trigger: Element) => void;
        onPaneSelect?: () => void;
        onStageSelect?: (stage: Stage, trigger: HTMLElement) => void;
        suppressStageFocusReveal?: boolean;
    }
    let {
        landscapes,
        stages,
        promotionConfigs,
        selectedStage,
        stageHref,
        stageReference,
        selectedPromotionId,
        highlightedStageKeys = [],
        onPromotionSelect,
        onPaneSelect,
        onStageSelect,
        suppressStageFocusReveal = false,
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
        const stageNodes: Node[] = graph.stages.map((placement) => {
            const key = stageKey(placement.landscapeId, placement.stage.name);
            return {
                ...defaults,
                data: {
                    href: stageHref(placement.stage),
                    landscapeName: placement.landscapeName,
                    onSelect: onStageSelect,
                    reference: stageReference(placement.stage),
                    selected:
                        (placement.stage.id === selectedStage?.id &&
                            placement.stage.landscapeId === selectedStage.landscapeId) ||
                        highlightedStageKeys.includes(key),
                    stage: placement.stage,
                    suppressFocusReveal: suppressStageFocusReveal,
                },
                height: CARD_HEIGHT,
                id: key,
                position: { x: columnX(placement.column), y: rowY(placement.row) },
                sourcePosition: Position.Right,
                targetPosition: Position.Left,
                type: "stage",
                width: CARD_WIDTH,
            };
        });
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

    const colorMode = $derived(mode.current === "dark" ? "dark" : "light");
    const edges = $derived.by((): Edge[] =>
        graph.edges.map((edge) => ({
            ariaLabel: `View promotion details from ${edge.sourceName} to ${edge.targetName}`,
            ariaRole: "button",
            class: "landscape-promotion-edge",
            data: { sourceName: edge.sourceName, targetName: edge.targetName },
            domAttributes: {
                onkeydown: (event: KeyboardEvent) => activateEdge(event, edge.id),
            },
            focusable: true,
            id: edge.id,
            selectable: false,
            selected: edge.id === selectedPromotionId,
            source: edge.sourceStageKey,
            target: edge.targetStageKey,
        })),
    );

    const onEdgeClick = ({ edge, event }: { edge: Edge; event: MouseEvent }): void => {
        onPromotionSelect?.(edge.id, event.currentTarget as Element);
    };

    const activateEdge = (event: KeyboardEvent, promotionId: string): void => {
        if (event.key !== "Enter" && event.key !== " ") {
            return;
        }
        event.preventDefault();
        onPromotionSelect?.(promotionId, event.currentTarget as Element);
    };

</script>

<div
    bind:clientWidth={canvasWidth}
    tabindex="-1"
    class="h-full min-h-0 min-w-0 bg-surface-canvas [&_.svelte-flow]:[--xy-background-color:var(--surface-canvas)] [&_.svelte-flow]:[--xy-controls-button-background-color:var(--surface-default)] [&_.svelte-flow]:[--xy-controls-button-color:var(--text-primary)] [&_.svelte-flow]:[--xy-controls-button-border-color:var(--border-subtle)]"
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
            edgesFocusable={true}
            onedgeclick={onEdgeClick}
            onpaneclick={onPaneSelect}
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

<style>
    /* Promotion edges are interactive: clicking one opens the promotion sheet.
       The wide invisible interaction path gets the pointer cursor, and the
       selected edge is highlighted. */
    :global(.landscape-promotion-edge .svelte-flow__edge-interaction),
    :global(.landscape-promotion-edge .svelte-flow__edge-path) {
        cursor: pointer;
    }

    :global(.landscape-promotion-edge.selected .svelte-flow__edge-path) {
        stroke: var(--border-strong, #b4851e);
        stroke-width: 2.5;
    }
</style>
