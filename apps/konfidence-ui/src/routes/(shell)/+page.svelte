<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { useProjects } from "$lib/projects/projects";
    import { dashboardTitle } from "$lib/dashboard";
    import { isEmbedded } from "$lib/shell/embedded";

    /**
     * Dashboard root. Redirects to the first project's landscape once projects
     * have loaded. If the user has no projects we show a minimal empty state
     * so the app never lands on a dead page.
     */
    const projects = useProjects();

    $effect(() => {
        if (!projects.selectedProjectId) return;
        const target = resolve("/(shell)/projects/[projectId]/landscape", {
            projectId: projects.selectedProjectId,
        });
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- `target` is already a resolved pathname; we may append `?embedded=1` for host integrations.
        void goto(isEmbedded(page.url) ? `${target}?embedded=1` : target, {
            replaceState: true,
        });
    });
</script>

<svelte:head>
    <title>{dashboardTitle}</title>
</svelte:head>

{#if projects.projects.length === 0}
    <section class="mx-auto flex max-w-[40rem] flex-col gap-3 px-6 py-10">
        <h1
            class="m-0 text-[color:var(--text-primary)] font-[weight:var(--weight-display)] [font-size:var(--text-h2)] [letter-spacing:var(--tracking-h2)]"
            data-testid="no-projects"
        >
            No projects
        </h1>
        <p class="m-0 text-[color:var(--text-secondary)] [font-size:var(--text-body)]">
            You are signed in but do not have access to any project yet.
        </p>
    </section>
{/if}
