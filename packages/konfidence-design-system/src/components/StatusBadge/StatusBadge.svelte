<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /**
         * Status identifier. Free-form on purpose: the API owns the
         * vocabulary (`healthy`, `deploying`, …) and this component is
         * a passive dispatcher — it appends the value to the badge
         * class so styling comes from the scoped `.badge--<status>`
         * rules below. Unknown values render as an unstyled `.badge`
         * with a `data-status` for debugging.
         */
        status: string;
        /** Whether to show the leading state dot. Defaults to `true`. */
        showDot?: boolean;
        /** Extra class names for layout tweaks. */
        class?: string;
        /**
         * Human-readable label. Required — a badge without visible text
         * violates the design-system auditability rule.
         */
        children: Snippet;
    }

    let { status, showDot = true, class: className, children }: Props = $props();

    const composedClass = $derived(
        className ? `badge badge--${status} ${className}` : `badge badge--${status}`,
    );
</script>

<span class={composedClass} data-status={status}>
    {#if showDot}
        <span class="dot" aria-hidden="true"></span>
    {/if}
    {@render children()}
</span>

<style>
    .badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold);
        padding: var(--badge-py) var(--badge-px) var(--badge-py) 8px;
        border-radius: var(--badge-radius);
        line-height: 1.4;
        white-space: nowrap;
        border: 1px solid transparent;
    }

    .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        flex-shrink: 0;
    }

    .badge--healthy {
        color: var(--status-healthy-fg);
        background: var(--status-healthy-bg);
    }
    .badge--healthy .dot {
        background: var(--status-healthy-solid);
    }

    .badge--warning {
        color: var(--status-warning-fg);
        background: var(--status-warning-bg);
    }
    .badge--warning .dot {
        background: var(--status-warning-solid);
    }

    .badge--degraded {
        color: var(--status-degraded-fg);
        background: var(--status-degraded-bg);
    }
    .badge--degraded .dot {
        background: var(--status-degraded-solid);
    }

    .badge--error {
        color: var(--status-error-fg);
        background: var(--status-error-bg);
    }
    .badge--error .dot {
        background: var(--status-error-solid);
    }

    .badge--promoting {
        color: var(--status-promoting-fg);
        background: var(--status-promoting-bg);
    }
    .badge--promoting .dot {
        background: var(--status-promoting-solid);
    }

    .badge--deploying {
        color: var(--status-deploying-fg);
        background: var(--status-deploying-bg);
    }
    .badge--deploying .dot {
        background: var(--status-deploying-solid);
    }

    .badge--queued {
        color: var(--status-queued-fg);
        background: var(--status-queued-bg);
    }
    .badge--queued .dot {
        background: var(--status-queued-solid);
    }
</style>
