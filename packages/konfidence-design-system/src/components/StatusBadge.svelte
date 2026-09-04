<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /**
         * Status identifier. Free-form on purpose: the API owns the
         * vocabulary (`healthy`, `deploying`, …) and this component is
         * a passive dispatcher — it appends the value to the badge
         * class so styling comes from `.badge--<status>` in
         * konfidence.custom.css. Unknown values render as an
         * unstyled `.badge` with a `data-status` for debugging.
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
