<script lang="ts">
    import type { Snippet } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { useSession } from "$lib/auth/session.svelte";
    import { useProjects as useProjectsQuery } from "$lib/queries.svelte";
    import { provideProjects } from "$lib/projects/projectContext";
    import { persistProjectPreferenceToSessionStorage, readLastProject } from "$lib/projects/persistProjectPreference";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";

    interface Props { children: Snippet; }
    let { children }: Props = $props();

    const session = useSession();
    const query = useProjectsQuery(() => session.user?.email ?? "", {
        enabled: () => session.status === "authenticated",
    });
    const projects = $derived(query.data ?? []);
    const status = $derived.by(() => {
        if (session.status !== "authenticated") { return "idle"; }
        if (query.loading) { return "loading"; }
        if (query.error) { return "error"; }
        return "ready";
    });

    const projectIdFromUrl = $derived(page.params.projectId as string | undefined);
    const selectedProject = $derived(
        status === "ready"
            ? projects.find((project) => project.id === projectIdFromUrl)
            : undefined,
    );

    const getEntryProject = () => {
        if (status !== "ready") {
            return undefined;
        }
        const rememberedId = session.user ? readLastProject(session.user.email) : undefined;
        return projects.find((project) => project.id === rememberedId)
            ?? (projects.length === 1 ? projects[0] : undefined);
    };

    const selectProject = async (projectId: string): Promise<void> => {
        if (status !== "ready" || !projects.some((project) => project.id === projectId)) {
            return;
        }
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- the shared helper resolves this typed application route.
        await goto(projectLandscapeUrl(projectId, isEmbedded(page.url)));
    };

    provideProjects({
        get error() { return query.error?.message; },
        getEntryProject,
        get projects() { return projects; },
        retry: () => query.reload(),
        selectProject,
        get selectedProject() { return selectedProject; },
        get status() { return status; },
    });

    $effect(() => {
        if (session.user && selectedProject) {
            persistProjectPreferenceToSessionStorage(session.user.email, selectedProject.id);
        }
    });
</script>

{@render children()}
