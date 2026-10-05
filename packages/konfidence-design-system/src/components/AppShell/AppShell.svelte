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
        /**
         * Layout mode.
         * - `"scroll"` (default): shell grows to fit content and the main
         *   region scrolls internally.
         * - `"canvas"`: shell is pinned to the viewport height and the main
         *   region hands vertical space to the page for full-height canvases
         *   (e.g. SvelteFlow). The page owns its own scrolling if needed.
         */
        layout?: "scroll" | "canvas";
    }

    let {
        topbar,
        sidebar,
        main,
        mainId = "app-shell-main",
        layout = "scroll",
    }: Props = $props();

    let drawerOpen = $state(false);
    const toggleDrawer = (): void => {
        drawerOpen = !drawerOpen;
    };
    const closeDrawer = (): void => {
        drawerOpen = false;
    };
</script>

<a
    class="skip-link absolute top-3 left-3 z-[500] -translate-y-[200%] rounded-base border border-outline-focus bg-surface-default py-1.5 px-2.5 text-compact text-content-primary focus-visible:translate-y-0"
    href={`#${mainId}`}
>Skip to main content</a>
<div
    class={[
        "app-shell grid grid-rows-[4px_56px_1fr]",
        layout === "canvas" ? "h-dvh min-h-0" : "min-h-dvh",
    ]}
>
    <Brandbar />
    {@render topbar({ toggleDrawer })}
    <div
        class={[
            "app-shell__body grid grid-cols-[224px_1fr] overflow-hidden max-md:grid-cols-1",
            layout === "canvas" && "min-h-0",
        ]}
    >
        <aside
            class={[
                "app-shell__sidebar flex flex-col overflow-y-auto [&>.sidebar]:flex-1",
                "max-md:fixed max-md:inset-[60px_0_0_0] max-md:z-[400] max-md:w-[260px] max-md:max-w-[80vw]",
                "max-md:transition-transform max-md:duration-[var(--motion-base)] max-md:ease-[cubic-bezier(var(--ease-out))]",
                "max-md:data-[open=false]:-translate-x-full max-md:data-[open=true]:translate-x-0",
            ]}
            data-open={drawerOpen ? "true" : "false"}
        >
            {@render sidebar({ closeDrawer })}
        </aside>
        <button
            type="button"
            class={[
                "app-shell__scrim hidden max-md:fixed max-md:inset-[60px_0_0_0] max-md:z-[399] max-md:block",
                "max-md:bg-[var(--scrim,rgba(0,0,0,0.4))] max-md:opacity-0 max-md:pointer-events-none",
                "max-md:transition-opacity max-md:duration-[var(--motion-base)] max-md:ease-[cubic-bezier(var(--ease))]",
                "max-md:data-[open=true]:opacity-100 max-md:data-[open=true]:pointer-events-auto",
            ]}
            data-open={drawerOpen ? "true" : "false"}
            aria-label="Close navigation"
            tabindex={drawerOpen ? 0 : -1}
            onclick={closeDrawer}
        ></button>
        <main
            class={[
                "app-shell__main min-w-0",
                layout === "canvas" ? "min-h-0 overflow-hidden" : "overflow-y-auto",
            ]}
            id={mainId}
        >
            {@render main()}
        </main>
    </div>
</div>
