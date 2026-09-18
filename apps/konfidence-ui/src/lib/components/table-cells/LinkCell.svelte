<script lang="ts">
    import type { Snippet } from "svelte";
    import type { ResolvedPathname } from "$app/types";

    interface Props {
        ariaLabel: string;
        children: Snippet;
        url: ResolvedPathname;
    }
    let { ariaLabel, children, url }: Props = $props();
</script>

<a href={url} aria-label={ariaLabel} onclick={(event) => event.stopPropagation()}>
    <span
        class="font-[family-name:var(--font-mono)] text-[length:var(--text-sm)] text-[color:var(--text-secondary)]"
        >{@render children()}</span
    >
</a>
