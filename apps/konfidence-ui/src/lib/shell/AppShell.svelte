<script lang="ts">
    import type { Snippet } from "svelte";
    import TopBar from "$lib/shell/TopBar.svelte";
    import SideNav from "$lib/shell/SideNav.svelte";
    import { closeDrawer, drawer } from "$lib/shell/sidebar.svelte";

    /**
     * Application shell. Grid layout of `brandbar / topbar / (sidebar | main)`.
     * Below the `md` breakpoint the sidebar becomes an off-canvas drawer
     * toggled by the hamburger in `TopBar`. All chrome hides in embedded mode;
     * see `(shell)/+layout.svelte`.
     */
    interface Props {
        children: Snippet;
    }

    let { children }: Props = $props();
</script>

<a class="skip-link" href="#app-shell-main">Skip to main content</a>
<div class="app-shell">
    <div class="brandbar" aria-hidden="true"></div>
    <TopBar />
    <div class="app-shell__body">
        <aside class="app-shell__sidebar" data-open={drawer.open ? "true" : "false"}>
            <SideNav />
        </aside>
        <button
            type="button"
            class="app-shell__scrim"
            data-open={drawer.open ? "true" : "false"}
            aria-label="Close navigation"
            tabindex={drawer.open ? 0 : -1}
            onclick={closeDrawer}
        ></button>
        <main class="app-shell__main" id="app-shell-main">
            {@render children()}
        </main>
    </div>
</div>
