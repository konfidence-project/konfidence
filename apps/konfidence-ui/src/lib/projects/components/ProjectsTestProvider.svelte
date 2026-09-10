<script lang="ts">
    import type { Project } from "$lib/projects/projectContext";
    import { provideProjects } from "$lib/projects/projectContext";
    import ProjectSelection from "$lib/projects/components/ProjectSelection.svelte";

    interface Props { projects: readonly Project[]; }
    let { projects }: Props = $props();
    provideProjects({
        getEntryProject: () => undefined,
        get projects() { return projects; },
        retry: async () => undefined,
        selectProject: async () => undefined,
        status: "ready",
    });
</script>

<ProjectSelection />
