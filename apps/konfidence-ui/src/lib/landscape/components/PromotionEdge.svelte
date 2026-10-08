<script lang="ts">
    import { BaseEdge, EdgeLabel, getSmoothStepPath } from "@xyflow/svelte";
    import type { EdgeProps } from "@xyflow/svelte";
    import type { PromotionEdgeStatus } from "$lib/landscape/landscapeGraph";
    import { promotionStatuses } from "$lib/landscape/promotionStatus";

    // SvelteFlow passes layout geometry plus our custom `data`. Everything is
    // optional on EdgeProps, so we read what we need and fall back gracefully.
    type Props = EdgeProps & {
        data?: { status?: PromotionEdgeStatus };
    };
    let {
        id,
        sourceX,
        sourceY,
        sourcePosition,
        targetX,
        targetY,
        targetPosition,
        data,
    }: Props = $props();

    // A smooth-step (orthogonal) path keeps the segment entering the target
    // handle horizontal regardless of how far the source sits above or below,
    // so the arrowhead always points cleanly right into the Left handle. A
    // bezier path would approach near-vertically for large row offsets and the
    // arrow would appear rotated/broken.
    const [path, labelX, labelY] = $derived(
        getSmoothStepPath({
            borderRadius: 40,
            sourcePosition,
            sourceX,
            sourceY,
            targetPosition,
            targetX,
            targetY,
        }),
    );

    const entry = $derived(
        data?.status === undefined ? undefined : promotionStatuses[data.status],
    );
    // A promoted edge takes its state colour; an edge whose config has never run
    // is drawn faint and dashed so it reads as "defined but idle".
    const edgeClass = $derived(
        entry === undefined
            ? "stroke-[var(--border-subtle)]! [stroke-dasharray:6_6] opacity-70"
            : entry.stroke,
    );
    const arrowClass = $derived(
        entry === undefined ? "fill-[var(--border-subtle)] opacity-70" : entry.arrowFill,
    );
    // Edge ids are JSON stage keys containing characters that are invalid in an
    // SVG fragment id / `url(#…)` reference. Encode to a safe token so the marker
    // actually resolves.
    const safeId = $derived(id.replaceAll(/[^a-zA-Z0-9_-]/g, "_"));
    const markerId = $derived(`promotion-arrow-${safeId}`);
</script>

<defs>
    <marker
        id={markerId}
        markerWidth="12"
        markerHeight="12"
        viewBox="0 0 12 12"
        refX="9"
        refY="6"
        markerUnits="userSpaceOnUse"
        orient="auto-start-reverse"
    >
        <path d="M2,2 L10,6 L2,10 z" class={arrowClass} />
    </marker>
</defs>
<BaseEdge {path} markerEnd={`url(#${markerId})`} class={edgeClass} />
{#if entry !== undefined}
    <EdgeLabel
        x={labelX}
        y={labelY}
        transparent
        class="pointer-events-none inline-flex items-center gap-1 whitespace-nowrap rounded-full px-1.5 py-px text-[0.6875rem] font-semibold leading-tight ring-2 ring-[var(--surface-canvas)] {entry.pill}"
    >
        <span class="size-1.5 shrink-0 rounded-full {entry.dot}" aria-hidden="true"></span>
        {entry.label}
    </EdgeLabel>
{/if}
