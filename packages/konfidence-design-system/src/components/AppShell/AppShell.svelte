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
