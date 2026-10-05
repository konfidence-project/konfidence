<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        children: Snippet;
    }
    let { children }: Props = $props();
</script>

<span
    class="font-mono text-compact text-content-secondary"
    >{@render children()}</span
>
