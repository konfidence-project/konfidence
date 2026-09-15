<script lang="ts">
    /* oxlint-disable eslint/id-length -- SvelteFlow positions use x and y coordinates. */
    import { useSvelteFlow } from "@xyflow/svelte";

    interface Props {
        canvasWidth: number;
        contentWidth: number;
        gap: number;
        minReadableZoom: number;
    }
    let { canvasWidth, contentWidth, gap, minReadableZoom }: Props = $props();

    const { setViewport } = useSvelteFlow();

    // Re-clamp the readable zoom whenever the visible canvas actually
    // resizes; the initial measurement is already applied through
    // `initialViewport` on <SvelteFlow>. Skips the first non-zero width so
    // we do not stomp on user pan/zoom on mount.
    let lastWidth = -1;
    $effect(() => {
        if (canvasWidth <= 0) {
            return;
        }
        if (lastWidth === -1) {
            lastWidth = canvasWidth;
            return;
        }
        if (canvasWidth === lastWidth) {
            return;
        }
        lastWidth = canvasWidth;
        const zoom = Math.max(
            minReadableZoom,
            Math.min(1, (canvasWidth - gap - gap) / contentWidth),
        );
        void setViewport({ x: gap, y: gap, zoom });
    });
</script>
