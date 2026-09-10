<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { useProjects } from "$lib/projects/projectContext";
    import { isEmbedded } from "$lib/shell/embedded";
    import { projectLandscapeUrl, projectsUrl } from "$lib/projects/url";

    const projects = useProjects();

    $effect(() => {
        if (projects.status !== "ready") {
            return;
        }
        const entryProject = projects.getEntryProject();
        const target = entryProject
            ? projectLandscapeUrl(entryProject.id, isEmbedded(page.url))
            : projectsUrl(isEmbedded(page.url));
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- the shared helper resolves this typed application route.
        void goto(target, { replaceState: true });
    });
</script>

<p role="status" class="p-6">Opening projects…</p>
