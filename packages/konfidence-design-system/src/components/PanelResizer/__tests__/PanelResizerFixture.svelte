<script lang="ts">
    import PanelResizer from "../PanelResizer.svelte";

    const DEFAULT_SIZE = 400;
    let { edge, initialSize = DEFAULT_SIZE, onResize }: {
        edge: "left" | "top";
        initialSize?: number;
        onResize?: (size: number) => void;
    } = $props();
    // This fixture deliberately seeds the bindable size once.
    // svelte-ignore state_referenced_locally
    let size = $state(initialSize);
</script>

<div class="relative h-96 w-96">
    <PanelResizer {edge} bind:size min={220} max={800} testid="resizer" {onResize} />
    <output data-testid="size">{size}</output>
</div>
