<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        children: Snippet;
    }
    let { children }: Props = $props();
</script>

<span
    class="font-ui-mono text-ui-sm text-content-secondary"
    >{@render children()}</span
>
