<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        children: Snippet;
    }
    let { children }: Props = $props();
</script>

<span
    class="font-[family-name:var(--font-mono)] text-[length:var(--text-sm)] text-[color:var(--text-secondary)]"
    >{@render children()}</span
>
