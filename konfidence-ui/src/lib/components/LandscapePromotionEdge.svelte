<script lang="ts">
    import {
        BaseEdge,
        EdgeLabel,
        type EdgeProps,
    } from "@xyflow/svelte";

    type PromotionEdgeData = {
        status?: "healthy" | "running" | "failed";
    };

    const {
        id,
        sourceX,
        sourceY,
        targetX,
        targetY,
        markerEnd,
        label,
        data,
    } = $props<EdgeProps & { data?: PromotionEdgeData }>();

    const status = $derived(data?.status ?? "healthy");
    const path = $derived.by(() => {
        const distance = Math.abs(targetX - sourceX);
        const curve = Math.max(120, distance * 0.55);

        return [
            `M ${sourceX} ${sourceY}`,
            `C ${sourceX + curve} ${sourceY}`,
            `${targetX - curve} ${targetY}`,
            `${targetX} ${targetY}`,
        ].join(" ");
    });
    const labelX = $derived((sourceX + targetX) / 2);
    const labelY = $derived((sourceY + targetY) / 2);
</script>

<BaseEdge {id} {path} {markerEnd} class={`promotion-edge ${status}`} />

{#if label}
    <EdgeLabel x={labelX} y={labelY} class={`promotion-edge-label ${status}`}>
        {label}
    </EdgeLabel>
{/if}

<style>
    :global(.promotion-edge) {
        stroke: var(--sapContent_ForegroundBorderColor);
        stroke-width: 1.6;
    }

    :global(.promotion-edge.healthy) {
        stroke: var(--sapPositiveElementColor);
    }

    :global(.promotion-edge.running) {
        stroke: var(--sapInformativeElementColor);
        stroke-dasharray: 8 6;
    }

    :global(.promotion-edge.failed) {
        stroke: var(--sapNegativeElementColor);
        stroke-dasharray: 4 5;
    }

    :global(.promotion-edge-label) {
        padding: 0.125rem 0.375rem;
        border: 1px solid var(--sapList_BorderColor);
        border-radius: 999px;
        background: var(--sapGroup_ContentBackground);
        color: var(--sapContent_LabelColor);
        box-shadow: var(--sapContent_Shadow0, 0 1px 2px rgba(0, 0, 0, 0.08));
        font-family: var(--sapFontMonospaceFamily, monospace);
        font-size: 0.6875rem;
        line-height: 1.2;
        white-space: nowrap;
    }

    :global(.promotion-edge-label.running) {
        border-color: var(--sapInformativeElementColor);
        color: var(--sapInformativeTextColor, var(--sapInformativeElementColor));
    }

    :global(.promotion-edge-label.failed) {
        border-color: var(--sapNegativeElementColor);
        color: var(--sapNegativeTextColor, var(--sapNegativeElementColor));
    }
</style>
