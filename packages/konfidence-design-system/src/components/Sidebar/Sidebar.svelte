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

<nav
    class="sidebar flex flex-col border-r border-[var(--border-subtle)] bg-[var(--surface-default)] py-4 px-3"
    aria-label="Primary"
>
    {#if mobileSwitcher}
        <div class="sidebar__project mb-4 block md:hidden">{@render mobileSwitcher()}</div>
    {/if}
    {@render children()}
    {#if footer}
        <div
            class="sidebar__footer mt-auto flex items-center gap-2 border-t border-[var(--border-subtle)] p-3 text-[length:var(--text-meta)] text-[var(--text-tertiary)]"
        >{@render footer()}</div>
    {/if}
</nav>

<style>
    /* overrides the nested <ProjectSwitcher>'s own classes when hosted in the mobile drawer */
    .sidebar__project :global(.project-switch) {
        width: 100%;
    }
    .sidebar__project :global(.project-switch__name) {
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        text-align: left;
    }
</style>
