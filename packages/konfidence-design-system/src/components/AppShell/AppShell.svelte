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

<a
    class="skip-link absolute top-3 left-3 z-[500] -translate-y-[200%] rounded-[var(--radius-md)] border border-[var(--border-focus)] bg-[var(--surface-default)] py-1.5 px-2.5 text-[length:var(--text-sm)] text-[var(--text-primary)] focus-visible:translate-y-0"
    href={`#${mainId}`}
>Skip to main content</a>
<div class="app-shell grid min-h-dvh grid-rows-[4px_56px_1fr]">
    <Brandbar />
    {@render topbar({ toggleDrawer })}
    <div class="app-shell__body grid grid-cols-[224px_1fr] overflow-hidden max-md:grid-cols-1">
        <aside class="app-shell__sidebar flex flex-col overflow-y-auto" data-open={drawerOpen ? "true" : "false"}>
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
        <main class="app-shell__main min-w-0 overflow-y-auto" id={mainId}>
            {@render main()}
        </main>
    </div>
</div>

<style>
    .app-shell__sidebar :global(> .sidebar) {
        flex: 1;
    }
    .app-shell__scrim {
        display: none;
    }
    @media (max-width: 767px) {
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
