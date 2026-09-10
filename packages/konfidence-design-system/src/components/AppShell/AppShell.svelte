<script lang="ts">
    import type { Snippet } from "svelte";
    import Brandbar from "../Brandbar/Brandbar.svelte";

    /**
     * Application shell. Grid layout of `brandbar / topbar / (sidebar | main)`.
     * Below the `md` breakpoint the sidebar becomes an off-canvas drawer,
     * toggled through the `toggleDrawer` snippet parameter passed into
     * `topbar`. Drawer state is owned internally — consumers do not need
     * to import anything.
     *
     * Consumers that want to close the drawer from inside the sidebar
     * (e.g. after picking a destination) receive `closeDrawer` as the
     * `sidebar` snippet's parameter.
     */
    interface Props {
        /**
         * Renders inside the topbar row. Receives `toggleDrawer` — when
         * called, opens/closes the mobile drawer. Wire it to your
         * `<TopBar>`'s `onHamburger` prop.
         */
        topbar: Snippet<[{ toggleDrawer: () => void }]>;
        /**
         * Renders inside the sidebar drawer. Receives `closeDrawer` —
         * pass it to each nav item's `onclick` so the drawer collapses
         * after a destination is picked.
         */
        sidebar: Snippet<[{ closeDrawer: () => void }]>;
        /** Main page content. */
        main: Snippet;
        /**
         * Id applied to the `<main>` element and pointed at by the skip
         * link. Overridable in case the app wants a shorter/aliased id.
         */
        mainId?: string;
    }

    let { topbar, sidebar, main, mainId = "app-shell-main" }: Props = $props();

    let drawerOpen = $state(false);
    const toggleDrawer = (): void => {
        drawerOpen = !drawerOpen;
    };
    const closeDrawer = (): void => {
        drawerOpen = false;
    };
</script>

<a class="skip-link" href={`#${mainId}`}>Skip to main content</a>
<div class="app-shell">
    <Brandbar />
    {@render topbar({ toggleDrawer })}
    <div class="app-shell__body">
        <aside class="app-shell__sidebar" data-open={drawerOpen ? "true" : "false"}>
            {@render sidebar({ closeDrawer })}
        </aside>
        <button
            type="button"
            class="app-shell__scrim"
            data-open={drawerOpen ? "true" : "false"}
            aria-label="Close navigation"
            tabindex={drawerOpen ? 0 : -1}
            onclick={closeDrawer}
        ></button>
        <main class="app-shell__main" id={mainId}>
            {@render main()}
        </main>
    </div>
</div>

<style>
    .skip-link {
        position: absolute;
        left: var(--space-3);
        top: var(--space-3);
        padding: 6px 10px;
        border-radius: var(--radius-md);
        background: var(--surface-default);
        color: var(--text-primary);
        border: 1px solid var(--border-focus);
        font-size: var(--text-sm);
        z-index: 500;
        transform: translateY(-200%);
    }
    .skip-link:focus-visible {
        transform: translateY(0);
    }

    .app-shell {
        display: grid;
        grid-template-rows: 4px 56px 1fr;
        min-height: 100dvh;
    }
    .app-shell__body {
        display: grid;
        grid-template-columns: 224px 1fr;
        overflow: hidden;
    }
    .app-shell__sidebar {
        display: flex;
        flex-direction: column;
        overflow-y: auto;
    }
    .app-shell__sidebar :global(> .sidebar) {
        flex: 1;
    }
    .app-shell__main {
        overflow-y: auto;
        min-width: 0;
    }
    /* applied to the child <IconButton> rendered by TopBar.svelte, not this component's own markup */
    :global(.app-shell__hamburger) {
        display: none;
    }
    .app-shell__scrim {
        display: none;
    }
    @media (max-width: 767px) {
        .app-shell__body {
            grid-template-columns: 1fr;
        }
        :global(.app-shell__hamburger) {
            display: flex;
        }
        .app-shell__sidebar {
            position: fixed;
            inset: 60px 0 0 0;
            width: 260px;
            max-width: 80vw;
            z-index: 400;
            transform: translateX(-100%);
            transition: transform var(--motion-base) cubic-bezier(var(--ease-out));
        }
        .app-shell__sidebar[data-open="true"] {
            transform: translateX(0);
        }
        .app-shell__scrim {
            position: fixed;
            inset: 60px 0 0 0;
            background: var(--scrim, rgba(0, 0, 0, 0.4));
            z-index: 399;
            display: block;
            opacity: 0;
            pointer-events: none;
            transition: opacity var(--motion-base) cubic-bezier(var(--ease));
        }
        .app-shell__scrim[data-open="true"] {
            opacity: 1;
            pointer-events: auto;
        }
    }
</style>
