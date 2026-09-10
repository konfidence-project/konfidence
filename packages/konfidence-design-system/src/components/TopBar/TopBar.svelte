<script lang="ts">
    import type { Snippet } from "svelte";
    import IconButton from "../IconButton/IconButton.svelte";

    interface Props {
        /** Brand logo slot (image or SVG). Rendered on the left. */
        logo: Snippet;
        /**
         * Optional project/context switcher slot. Below the `md`
         * breakpoint the shell moves this slot into the sidebar drawer
         * — the topbar hides its copy via `.topbar__proj-switch` CSS.
         */
        switcher?: Snippet;
        /**
         * Trailing action slots — avatar menu, notification bell, etc.
         * Rendered inside `.topbar__actions`.
         */
        actions: Snippet;
        /**
         * When set, renders a leading hamburger button that calls the
         * callback on click. The button is only visible below the `md`
         * breakpoint (styled via `.app-shell__hamburger`).
         */
        onHamburger?: () => void;
    }

    let { logo, switcher, actions, onHamburger }: Props = $props();
</script>

<div class="topbar">
    {#if onHamburger}
        <IconButton
            icon="menu2"
            ariaLabel="Toggle navigation"
            class="app-shell__hamburger"
            data-testid="drawer-toggle"
            onclick={onHamburger}
        />
    {/if}
    {@render logo()}
    {#if switcher}
        <div class="topbar__proj-switch">
            {@render switcher()}
        </div>
    {/if}
    <div class="topbar__actions">
        {@render actions()}
    </div>
</div>

<style>
    .topbar {
        display: flex;
        align-items: center;
        gap: var(--space-4);
        height: 56px;
        padding: 0 var(--space-4);
        background: var(--surface-default);
        border-bottom: 1px solid var(--border-subtle);
    }
    /* applied by the consumer's `logo` snippet content, not this component's own markup */
    :global(.topbar__logo) {
        height: 22px;
        flex-shrink: 0;
    }
    .topbar__actions {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        margin-left: auto;
    }
    /* hidden on mobile; Sidebar.svelte's `.sidebar__project` takes over there (see AppShell's drawer media query) */
    .topbar__proj-switch {
        display: flex;
    }
    @media (max-width: 767px) {
        .topbar__proj-switch {
            display: none;
        }
    }
</style>
