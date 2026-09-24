<script lang="ts">
    import type { Snippet } from "svelte";
    import { page } from "$app/state";
    import { useProjects } from "$lib/projects/projectContext";
    import { isEmbedded } from "$lib/shell/embedded";
    import { projectsUrl } from "$lib/projects/url";
    import { Link } from "@konfidence/design-system/components";

    interface Props { children: Snippet; }
    let { children }: Props = $props();
    const projects = useProjects();
    const available = $derived(projects.status === "ready" && projects.selectedProject !== undefined);
</script>

{#if available}
    {@render children()}
{:else if projects.status === "ready"}
    <section class="mx-auto max-w-[40rem] p-6" aria-labelledby="project-unavailable-title">
        <h1 id="project-unavailable-title">Project not found or unavailable</h1>
        <p>This project does not exist or you do not have access to it.</p>
        <Link href={projectsUrl(isEmbedded(page.url))}>Back to project selection</Link>
    </section>
{/if}
