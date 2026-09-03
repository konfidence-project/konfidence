<script lang="ts">
    import type { Snippet } from "svelte";

    /**
     * Named domain state. Auditability rule: badges are always identified
     * by name (text + colour), never by colour alone.
     */
    type Status =
        | "healthy"
        | "warning"
        | "degraded"
        | "error"
        | "promoting"
        | "deploying"
        | "queued";

    interface Props {
        /** Semantic status. Drives colour + surface tokens. */
        status: Status;
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
