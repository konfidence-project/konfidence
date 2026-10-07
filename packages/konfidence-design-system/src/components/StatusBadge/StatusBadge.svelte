<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /**
         * Status identifier. Free-form on purpose: the API owns the
         * vocabulary (`healthy`, `deploying`, …) and this component is
         * a passive dispatcher — it appends the value to the badge
         * class so styling comes from the known-status utility map.
         * Unknown values render without a mapped status tone, as `.badge`
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

    const STATUS_CLASSES: Record<string, { badge: string; dot: string }> = {
        degraded: { badge: "text-status-degraded-fg bg-status-degraded-bg", dot: "bg-status-degraded-solid" },
        deploying: { badge: "text-status-deploying-fg bg-status-deploying-bg", dot: "bg-status-deploying-solid" },
        error: { badge: "text-status-error-fg bg-status-error-bg", dot: "bg-status-error-solid" },
        healthy: { badge: "text-status-healthy-fg bg-status-healthy-bg", dot: "bg-status-healthy-solid" },
        promoting: { badge: "text-status-promoting-fg bg-status-promoting-bg", dot: "bg-status-promoting-solid" },
        queued: { badge: "text-status-queued-fg bg-status-queued-bg", dot: "bg-status-queued-solid" },
        warning: { badge: "text-status-warning-fg bg-status-warning-bg", dot: "bg-status-warning-solid" },
    };
</script>

<span class={[
    "badge inline-flex items-center gap-1.5 whitespace-nowrap rounded-[var(--badge-radius)] border border-transparent pt-[var(--badge-py)] pb-[var(--badge-py)] pr-[var(--badge-px)] pl-2 text-meta font-semibold leading-[1.4]",
    `badge--${status}`,
    STATUS_CLASSES[status]?.badge,
    className,
]} data-status={status}>
    {#if showDot}
        <span class={["dot size-2 shrink-0 rounded-full", STATUS_CLASSES[status]?.dot]} aria-hidden="true"></span>
    {/if}
    {@render children()}
</span>
