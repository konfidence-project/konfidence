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

    const composedClass = $derived(className ? `nav-group mb-5 ${className}` : "nav-group mb-5");
</script>

<div class={composedClass}>
    {#if label}
        <div
            class="nav-group__label mb-2 px-3 text-[10px] font-bold tracking-[0.09em] text-[var(--text-tertiary)] uppercase"
        >{label}</div>
    {/if}
    {@render children()}
</div>
