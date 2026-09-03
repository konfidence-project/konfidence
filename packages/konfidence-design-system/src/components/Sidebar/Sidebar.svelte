<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /** Default slot for `<NavGroup>` children. */
        children: Snippet;
        /**
         * Slot rendered inside `.sidebar__project` — visible only in the
         * mobile drawer (< md), where the project switcher moves from
         * the topbar into the drawer.
         */
        mobileSwitcher?: Snippet;
        /**
         * Slot rendered inside `.sidebar__footer` — a general
         * bottom-of-sidebar area for status text, version badges, or a
         * region label. Rendered only when the snippet is supplied.
         */
        footer?: Snippet;
    }

    let { children, mobileSwitcher, footer }: Props = $props();
</script>

<nav class="sidebar" aria-label="Primary">
    {#if mobileSwitcher}
        <div class="sidebar__project">{@render mobileSwitcher()}</div>
    {/if}
    {@render children()}
    {#if footer}
        <div class="sidebar__footer">{@render footer()}</div>
    {/if}
</nav>
