<script lang="ts">
    import { page } from "$app/state";
    import { useProjects } from "$lib/projects/projectContext";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";

    const projectContext = useProjects();
    const embedded = $derived(isEmbedded(page.url));
</script>

<section
    class="mx-auto max-w-[40rem] [padding:var(--space-8)] text-content-primary"
    aria-labelledby="project-selection-title"
>
    <h1 class="[font-size:var(--font-size-h2)] [margin-block:0_var(--space-3)]" id="project-selection-title">Choose a project</h1>
    {#if projectContext.projects.length === 0}
        <p data-testid="no-projects">You do not have access to any projects.</p>
    {:else}
        <p>Select the project you want to open.</p>
        <ul class="grid list-none gap-[var(--space-3)] p-0">
            {#each projectContext.projects as project (project.id)}
                <li>
                    <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the shared helper resolves this typed application route. -->
                    <a href={projectLandscapeUrl(project.id, embedded)}
                        class="block rounded-base border border-outline-default p-[var(--space-4)] text-inherit no-underline hover:bg-surface-subtle focus-visible:bg-surface-subtle"
                    >{project.name}</a>
                </li>
            {/each}
        </ul>
    {/if}
</section>
