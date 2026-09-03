<script lang="ts">
    import type { Snippet } from "svelte";
    import { beforeNavigate, goto } from "$app/navigation";
    import { page } from "$app/state";
    import AppShell from "$lib/shell/AppShell.svelte";
    import { provideProjects } from "$lib/projects/projects";
    import { EMBEDDED_QUERY, EMBEDDED_ON, isEmbedded } from "$lib/shell/embedded";
    import type { LayoutProps } from "./$types";

    /**
     * Authenticated shell layout.
     *
     * - Provides the projects context loaded by `+layout.ts`.
     * - Selects a project id from the URL when present, else defaults to the
     *   first project. Both cases feed the shell's project switcher and side
     *   nav destinations.
     * - Renders `<AppShell>` unless the URL carries `?embedded=1`, in which
     *   case the page owns the full viewport (host-integration mode).
     * - Keeps `?embedded=1` sticky across client-side navigation via
     *   `beforeNavigate` so internal `<a>` and `goto()` calls preserve it.
     */
    interface Props {
        children: Snippet;
        data: LayoutProps["data"];
    }

    let { children, data }: Props = $props();

    const projectIdFromUrl = $derived(
        page.params.projectId as string | undefined,
    );
    const selectedProjectId = $derived(
        projectIdFromUrl ?? data.projects[0]?.id ?? "",
    );

    provideProjects({
        get projects() {
            return data.projects;
        },
        get selectedProjectId() {
            return selectedProjectId;
        },
    });

    const embedded = $derived(isEmbedded(page.url));

    beforeNavigate((navigation) => {
        if (!embedded || !navigation.to) return;
        const target = navigation.to.url;
        if (target.searchParams.get(EMBEDDED_QUERY) === EMBEDDED_ON) return;
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
        {@render children()}
    </AppShell>
{/if}
