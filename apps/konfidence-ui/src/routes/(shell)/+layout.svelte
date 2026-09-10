<script lang="ts">
    import type { Snippet } from "svelte";
    import { beforeNavigate, goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { AppShell, TopBar } from "@konfidence/design-system/components";
    import { isDarkMode, themeStore } from "$lib/theme";
    import { provideProjects } from "$lib/projects/projects";
    import ProjectSelector from "$lib/shell/ProjectSelector.svelte";
    import SideNav from "$lib/shell/SideNav.svelte";
    import UserMenu from "$lib/shell/UserMenu.svelte";
    import { EMBEDDED_ON, EMBEDDED_QUERY, isEmbedded } from "$lib/shell/embedded";
    import type { LayoutProps } from "./$types";

    /**
     * Authenticated shell layout.
     *
     * - Provides the projects context loaded by `+layout.ts`.
     * - Selects a project id from the URL when present, else defaults to the
     *   first project. Both cases feed the shell's project switcher and side
     *   nav destinations.
     * - Renders `<AppShell>` (from `@konfidence/design-system`) unless the URL
     *   carries `?embedded=1`, in which case the page owns the full viewport
     *   (host-integration mode).
     * - Keeps `?embedded=1` sticky across client-side navigation via
     *   `beforeNavigate` so internal `<a>` and `goto()` calls preserve it.
     *   Skips navigations that will unload the page (external links, full
     *   page reloads) since `goto()` can't target those.
     */
    interface Props {
        children: Snippet;
        data: LayoutProps["data"];
    }

    let { children, data }: Props = $props();

    const projectIdFromUrl = $derived(page.params.projectId as string | undefined);
    const selectedProjectId = $derived(projectIdFromUrl ?? data.projects[0]?.id ?? "");

    provideProjects({
        get projects() {
            return data.projects;
        },
        get selectedProjectId() {
            return selectedProjectId;
        },
    });

    const embedded = $derived(isEmbedded(page.url));

    const isDark = $derived(isDarkMode(themeStore.mode));
    const logoSrc = $derived(isDark ? "/logos/logo-dark.svg" : "/logos/logo-light.svg");

    beforeNavigate((navigation) => {
        if (!embedded || !navigation.to || navigation.willUnload) {
            return;
        }
        const target = navigation.to.url;
        if (target.searchParams.get(EMBEDDED_QUERY) === EMBEDDED_ON) {
            return;
        }
        navigation.cancel();
        const next = new globalThis.URL(target);
        next.searchParams.set(EMBEDDED_QUERY, EMBEDDED_ON);
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- the target URL is already resolved by SvelteKit; we only re-attach the ?embedded=1 flag before navigating.
        void goto(next, { replaceState: navigation.type === "leave" });
    });
</script>

{#if embedded}
    <main class="min-h-dvh" data-testid="embedded-main">
        {@render children()}
    </main>
{:else}
    <AppShell>
        {#snippet topbar({ toggleDrawer })}
            <TopBar onHamburger={toggleDrawer}>
                {#snippet logo()}
                    <a href={resolve("/")} aria-label="Konfidence home" data-testid="brand-home">
                        <img class="topbar__logo" src={logoSrc} alt="Konfidence" />
                    </a>
                {/snippet}
                {#snippet switcher()}
                    <ProjectSelector />
                {/snippet}
                {#snippet actions()}
                    <UserMenu />
                {/snippet}
            </TopBar>
        {/snippet}

        {#snippet sidebar({ closeDrawer })}
            <SideNav {closeDrawer} />
        {/snippet}

        {#snippet main()}
            {@render children()}
        {/snippet}
    </AppShell>
{/if}
