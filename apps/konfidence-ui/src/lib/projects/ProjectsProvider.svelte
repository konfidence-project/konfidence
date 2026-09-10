<script lang="ts">
    import type { Snippet } from "svelte";
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { useSession } from "$lib/auth/session.svelte";
    import { getApiClient } from "$lib/konfidence-api/client-instance";
    import { ProjectsStore } from "$lib/projects/projects.svelte";
    import { provideProjects } from "$lib/projects/projectContext";
    import { persistProjectPreferenceToSessionStorage, readLastProject } from "$lib/projects/persistProjectPreference";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";

    interface Props { children: Snippet; }
    let { children }: Props = $props();

    const session = useSession();
    const projectStore = new ProjectsStore(getApiClient());

    const projectIdFromUrl = $derived(page.params.projectId as string | undefined);
    const selectedProject = $derived(
        projectStore.status === "ready"
            ? projectStore.projects.find((project) => project.id === projectIdFromUrl)
            : undefined,
    );

    const getEntryProject = () => {
        if (projectStore.status !== "ready") {
            return undefined;
        }
        const rememberedId = session.user ? readLastProject(session.user.email) : undefined;
        return projectStore.projects.find((project) => project.id === rememberedId)
            ?? (projectStore.projects.length === 1 ? projectStore.projects[0] : undefined);
    };

    const selectProject = async (projectId: string): Promise<void> => {
        if (projectStore.status !== "ready" || !projectStore.projects.some((project) => project.id === projectId)) {
            return;
        }
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- the shared helper resolves this typed application route.
        await goto(projectLandscapeUrl(projectId, isEmbedded(page.url)));
    };

    provideProjects({
        get error() { return projectStore.error; },
        getEntryProject,
        get projects() { return projectStore.projects; },
        retry: () => projectStore.refresh(),
        selectProject,
        get selectedProject() { return selectedProject; },
        get status() { return projectStore.status; },
    });

    onMount(() => {
        void projectStore.refresh();
    });

    $effect(() => {
        if (session.user && selectedProject) {
            persistProjectPreferenceToSessionStorage(session.user.email, selectedProject.id);
        }
    });
</script>

{@render children()}
