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

<section id={id} class="gallery-section" data-status={status}>
    <header class="gallery-section__header">
        <div class="gallery-section__titles">
            <h2 class="gallery-section__title">{title}</h2>
            {#if subtitle}
                <p class="gallery-section__subtitle">{subtitle}</p>
            {/if}
        </div>
        <span class="gallery-section__badge" data-status={status}>{STATUS_LABEL[status]}</span>
    </header>
    <div class="gallery-section__body">
        {@render children()}
    </div>
</section>

<style>
    .gallery-section {
        margin: 0 0 48px;
        padding: 24px 24px 28px;
        border: 1px solid var(--border-subtle);
        border-radius: var(--radius-lg, 14px);
        background: var(--surface-card, var(--surface-default));
        box-shadow: var(--shadow-xs);
        /*
         * Offset in-page anchor jumps so section headings clear the
         * sticky brandbar + topbar (~90px total). Kept slightly loose
         * to leave breathing room above the heading.
         */
        scroll-margin-top: 96px;
    }

    .gallery-section__header {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 16px;
        margin-bottom: 20px;
        padding-bottom: 12px;
        border-bottom: 1px solid var(--border-subtle);
    }

    .gallery-section__title {
        margin: 0;
        font-size: var(--text-h2);
        font-weight: var(--weight-display, 600);
        color: var(--text-primary);
        letter-spacing: -0.5px;
    }

    .gallery-section__subtitle {
        margin: 4px 0 0;
        font-size: var(--text-sm);
        color: var(--text-secondary);
        max-width: 720px;
    }

    .gallery-section__badge {
        flex-shrink: 0;
        display: inline-flex;
        align-items: center;
        padding: 4px 10px;
        border-radius: var(--radius-pill, 999px);
        border: 1px solid var(--border-subtle);
        background: var(--surface-subtle);
        color: var(--text-secondary);
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        letter-spacing: 0.02em;
        text-transform: uppercase;
    }

    .gallery-section__badge[data-status="implemented"] {
        background: var(--status-healthy-bg, rgba(23, 178, 106, 0.1));
        color: var(--status-healthy-fg, #067647);
        border-color: transparent;
    }

    .gallery-section__badge[data-status="partial"] {
        background: var(--status-warning-bg, rgba(247, 144, 9, 0.12));
        color: var(--status-warning-fg, #b54708);
        border-color: transparent;
    }

    .gallery-section__badge[data-status="placeholder"] {
        background: var(--surface-sunken);
        color: var(--text-tertiary, var(--text-secondary));
    }
</style>
