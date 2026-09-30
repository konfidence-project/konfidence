<script lang="ts">
    import type { Project } from "$lib/konfidence-api/types";
    import { provideProjects } from "$lib/projects/projectContext";
    import ProjectsPage from "../+page.svelte";

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

<ProjectsPage />
