<script lang="ts">
    import type { Snippet } from "svelte";
    import Playground from "../playground/Playground.svelte";

    interface Props {
        title?: string;
        description?: string;
        children: Snippet;
        source?: string;
        previewHeight?: number;
    }

    let { title, description, children, source, previewHeight }: Props = $props();
</script>

<figure class="sample m-0 flex flex-col gap-3 rounded-base border border-dashed border-outline-subtle bg-surface-default p-4">
    {#if title || description}
        <figcaption class="sample__caption flex flex-col gap-0.5">
            {#if title}<span class="sample__title text-compact font-semibold text-content-primary">{title}</span>{/if}
            {#if description}<span class="sample__desc text-meta text-content-secondary">{description}</span>{/if}
        </figcaption>
    {/if}
    <div class="sample__body flex flex-wrap items-center gap-3">
        {#if source}<Playground {source} {title} {previewHeight} />{:else}{@render children()}{/if}
    </div>
</figure>
