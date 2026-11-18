<script lang="ts">
    import { getContext } from "svelte";
    import type { Snippet } from "svelte";
    import Readme from "./Readme.svelte";
    import ComponentAnchor from "./ComponentAnchor.svelte";
    import { CATALOG_ENTRY } from "./catalog-entry-context.ts";
    import type { CurrentCatalogEntry } from "./catalog-entry-context.ts";

    interface Props {
        id: string;
        title: string;
        subtitle?: string;
        status?: "implemented" | "partial" | "placeholder";
        children: Snippet;
    }

    let { id, title, subtitle, status = "implemented", children }: Props = $props();
    const currentEntry = getContext<CurrentCatalogEntry | undefined>(CATALOG_ENTRY);
    const entry = $derived(currentEntry?.());
    const readme = $derived(entry?.readme);

    const STATUS_LABEL: Record<NonNullable<Props["status"]>, string> = {
        implemented: "Implemented",
        partial: "Partial",
        placeholder: "Planned",
    };
</script>

<section id={id} class="gallery-section mb-12 scroll-mt-24 rounded-container border border-outline-subtle bg-surface-card px-6 pt-6 pb-7 shadow-elevation-xs" data-status={status}>
    <header class="mb-5 flex items-start justify-between gap-4 border-b border-outline-subtle pb-3">
        <div class="min-w-0 [overflow-wrap:anywhere]">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
                <h2 class="m-0 min-w-0 text-h2 font-display tracking-[-0.5px] text-content-primary">{title}</h2>
                {#if entry?.name === title}<ComponentAnchor {entry} />{/if}
            </div>
            {#if subtitle}
                <p class="mt-1 max-w-[720px] text-compact text-content-secondary">{subtitle}</p>
            {/if}
        </div>
        <span class={[
            "inline-flex shrink-0 items-center rounded-full border px-2.5 py-1 text-meta font-semibold tracking-[0.02em] uppercase",
            status === "implemented" && "border-transparent bg-status-healthy-bg text-status-healthy-fg",
            status === "partial" && "border-transparent bg-status-warning-bg text-status-warning-fg",
            status === "placeholder" && "border-outline-subtle bg-surface-sunken text-content-tertiary",
        ]} data-status={status}>{STATUS_LABEL[status]}</span>
    </header>
    <div>
        {#if readme}<Readme name={readme} />{/if}
        {@render children()}
    </div>
</section>
