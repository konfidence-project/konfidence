<script lang="ts">
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import ProjectSelection from "$lib/projects/components/ProjectSelection.svelte";
    import { useProjects } from "$lib/projects/projectContext";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";

    const projectContext = useProjects();
    onMount(() => {
        if (projectContext.status === "ready" && projectContext.projects.length === 1) {
            // eslint-disable-next-line svelte/no-navigation-without-resolve -- the shared helper resolves this typed application route.
            void goto(projectLandscapeUrl(projectContext.projects[0].id, isEmbedded(page.url)), { replaceState: true });
        }
    });
</script>

<svelte:head><title>Projects | Konfidence</title></svelte:head>
{#if projectContext.status === "ready"}
    {#if projectContext.projects.length === 1}
        <p class="p-6" role="status">Opening {projectContext.projects[0].name}…</p>
    {:else}
        <ProjectSelection />
    {/if}
{/if}
