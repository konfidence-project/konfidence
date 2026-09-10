<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /**
         * Uppercase group label rendered above the items. Purely
         * decorative — assistive tech reaches items directly via the
         * enclosing `<nav>`.
         */
        label?: string;
        /** Extra class names for layout tweaks. */
        class?: string;
        /** `<NavItem>` children. */
        children: Snippet;
    }

    let { label, class: className, children }: Props = $props();

    const composedClass = $derived(className ? `nav-group ${className}` : "nav-group");
</script>

<div class={composedClass}>
    {#if label}
        <div class="nav-group__label">{label}</div>
    {/if}
    {@render children()}
</div>

<style>
    .nav-group {
        margin-bottom: var(--space-5);
    }
    .nav-group__label {
        font-size: 10px;
        font-weight: var(--weight-bold);
        letter-spacing: 0.09em;
        text-transform: uppercase;
        color: var(--text-tertiary);
        padding: 0 var(--space-3);
        margin-bottom: var(--space-2);
    }
</style>
