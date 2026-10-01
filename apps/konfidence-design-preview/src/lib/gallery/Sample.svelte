<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        title?: string;
        description?: string;
        children: Snippet;
    }

    let { title, description, children }: Props = $props();
</script>

<figure class="sample">
    {#if title || description}
        <figcaption class="sample__caption">
            {#if title}<span class="sample__title">{title}</span>{/if}
            {#if description}<span class="sample__desc">{description}</span>{/if}
        </figcaption>
    {/if}
    <div class="sample__body">
        {@render children()}
    </div>
</figure>

<style>
    .sample {
        margin: 0;
        padding: 16px;
        border: 1px dashed var(--border-subtle);
        border-radius: 10px;
        background: var(--surface-default);
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .sample__caption {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .sample__title {
        font-size: var(--text-sm);
        font-weight: var(--weight-semibold, 600);
        color: var(--text-primary);
    }

    .sample__desc {
        font-size: var(--text-meta);
        color: var(--text-secondary);
    }

    .sample__body {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 12px;
    }
</style>
