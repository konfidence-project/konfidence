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

<style>
    .sidebar {
        background: var(--surface-default);
        border-right: 1px solid var(--border-subtle);
        padding: var(--space-4) var(--space-3);
        display: flex;
        flex-direction: column;
    }
    .sidebar__footer {
        margin-top: auto;
        padding: var(--space-3);
        border-top: 1px solid var(--border-subtle);
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: var(--text-meta);
        color: var(--text-tertiary);
    }

    /* hidden on desktop; TopBar.svelte's `.topbar__proj-switch` renders the switcher there instead */
    .sidebar__project {
        display: none;
        margin-bottom: var(--space-4);
    }
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
    @media (max-width: 767px) {
        .sidebar__project {
            display: block;
        }
    }
</style>
