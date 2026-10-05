<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        id: string;
        title: string;
        subtitle?: string;
        status?: "implemented" | "partial" | "placeholder";
        children: Snippet;
    }

    let { id, title, subtitle, status = "implemented", children }: Props = $props();

    const STATUS_LABEL: Record<NonNullable<Props["status"]>, string> = {
        implemented: "Implemented",
        partial: "Partial",
        placeholder: "Not yet implemented",
    };
</script>

<section id={id} class="gallery-section mb-12 scroll-mt-24 rounded-container border border-outline-subtle bg-surface-card px-6 pt-6 pb-7 shadow-elevation-xs" data-status={status}>
    <header class="mb-5 flex items-start justify-between gap-4 border-b border-outline-subtle pb-3">
        <div>
            <h2 class="m-0 text-h2 font-display tracking-[-0.5px] text-content-primary">{title}</h2>
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
        {@render children()}
    </div>
</section>
