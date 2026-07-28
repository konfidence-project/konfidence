<script lang="ts">
    import FolderIcon from "@lucide/svelte/icons/folder";
    import FolderSearchIcon from "@lucide/svelte/icons/folder-search";
    import ArrowRightIcon from "@lucide/svelte/icons/arrow-right";
    import { resolve } from "$app/paths";
    import type { PageProps } from "./$types";

    const { data }: PageProps = $props();
</script>

<svelte:head><title>Projects | Konfidence</title></svelte:head>

<section class="mx-auto grid w-full max-w-240 content-start gap-6 p-[clamp(1.25rem,4vw,3rem)]" aria-labelledby="projects-title">
    <header class="grid gap-1.5">
        <span class="text-xs font-bold tracking-[0.1em] text-app-accent-strong uppercase">Delivery workspaces</span>
        <h1 id="projects-title" class="m-0 text-[clamp(1.6rem,3vw,2.25rem)]">Projects</h1>
        <p class="m-0 text-app-muted">Select a project to inspect its delivery landscape.</p>
    </header>

    {#if data.projects.length === 0}
        <div class="card grid justify-items-center gap-3 border border-dashed border-app-border bg-app-card px-6 py-16 text-center">
            <FolderSearchIcon class="size-12 text-app-accent" aria-hidden="true" />
            <h2 class="m-0">No projects available</h2>
            <p class="m-0 text-app-muted">Your account does not currently have access to any projects.</p>
        </div>
    {:else}
        <ul class="card m-0 list-none overflow-hidden border border-app-border bg-app-card p-0 shadow-xl" aria-label="Projects">
            {#each data.projects as project (project.id)}
                <li class="border-t border-app-border first:border-t-0">
                    <a class="grid min-h-20 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 px-4 py-3 text-app-text no-underline hover:bg-app-accent/7" href={resolve(`/projects/${project.id}/landscape`)}>
                        <span class="grid size-10 place-items-center rounded-[0.6rem] bg-app-accent/13 text-app-accent-strong"><FolderIcon class="size-[1.15rem]" aria-hidden="true" /></span>
                        <span class="grid min-w-0 gap-1">
                            <strong class="text-base">{project.name}</strong>
                            <small class="overflow-wrap-anywhere text-app-muted">Project ID: {project.id}</small>
                        </span>
                        <ArrowRightIcon class="size-[1.15rem] text-app-muted" aria-hidden="true" />
                    </a>
                </li>
            {/each}
        </ul>
    {/if}
</section>
