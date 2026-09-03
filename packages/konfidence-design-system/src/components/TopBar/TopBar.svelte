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
            icon="menu"
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
