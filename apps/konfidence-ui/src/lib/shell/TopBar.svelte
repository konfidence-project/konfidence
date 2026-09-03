<script lang="ts">
    import { resolve } from "$app/paths";
    import ProjectSelector from "$lib/shell/ProjectSelector.svelte";
    import UserMenu from "$lib/shell/UserMenu.svelte";
    import { toggleDrawer } from "$lib/shell/sidebar.svelte";
    import { themeStore } from "@konfidence/design-system/theme";

    const logoSrc = $derived(
        themeStore.current === "konfidence-dark"
            ? "/logos/logo-dark.svg"
            : "/logos/logo-light.svg",
    );
</script>

<div class="topbar">
    <button
        type="button"
        class="icon-btn app-shell__hamburger"
        aria-label="Toggle navigation"
        data-testid="drawer-toggle"
        onclick={toggleDrawer}
    >
        <span aria-hidden="true">≡</span>
    </button>
    <a href={resolve("/")} aria-label="Konfidence home" data-testid="brand-home">
        <img class="topbar__logo" src={logoSrc} alt="Konfidence" />
    </a>
    <div class="topbar__proj-switch">
        <ProjectSelector />
    </div>
    <div class="topbar__actions">
        <UserMenu />
    </div>
</div>
