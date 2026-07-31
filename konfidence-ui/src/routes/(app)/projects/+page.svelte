<script lang="ts">
    import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
    import ArrowRightIcon from "@lucide/svelte/icons/arrow-right";
    import * as Card from "$lib/components/ui/card/index.js";
    import * as Empty from "$lib/components/ui/empty/index.js";
    import { resolve } from "$app/paths";
    import type { PageProps } from "./$types";

    const { data }: PageProps = $props();
</script>

<section
    class="mx-auto grid w-full max-w-[58rem] gap-7 px-[clamp(1rem,4vw,2rem)] py-[clamp(1.5rem,5vw,3.5rem)]"
    aria-labelledby="projects-title"
>
    <header class="grid gap-[0.35rem]">
        <h1 class="text-[clamp(1.65rem,4vw,2.25rem)] font-semibold tracking-[-0.03em]" id="projects-title">
            Projects
        </h1>
        <p class="text-muted-foreground">Select a project to inspect its delivery landscape.</p>
    </header>

    {#if data.projects.length === 0}
        <Empty.Root class="border bg-card">
            <Empty.Media variant="icon"><FolderKanbanIcon /></Empty.Media>
            <Empty.Header>
                <Empty.Title>No projects available</Empty.Title>
                <Empty.Description>
                    Your account does not currently have access to any projects.
                </Empty.Description>
            </Empty.Header>
        </Empty.Root>
    {:else}
        <ul class="grid list-none gap-[0.65rem] p-0" aria-label="Projects">
            {#each data.projects as project (project.id)}
                <li>
                    <Card.Root
                        class="p-0 transition-[border-color,transform,box-shadow] duration-120 hover:-translate-y-px hover:border-primary hover:shadow-[0_0.5rem_1.5rem_color-mix(in_oklch,var(--foreground)_8%,transparent)]"
                    >
                        <a
                            class="flex min-h-19 items-center justify-between gap-4 px-5 py-4 text-inherit no-underline"
                            href={resolve(`/projects/${project.id}/landscape`)}
                        >
                            <span class="grid gap-1">
                                <strong>{project.name}</strong>
                                <small class="text-[0.8rem] text-muted-foreground">
                                    Project ID: {project.id}
                                </small>
                            </span>
                            <ArrowRightIcon class="w-[1.15rem] text-primary" aria-hidden="true" />
                        </a>
                    </Card.Root>
                </li>
            {/each}
        </ul>
    {/if}
</section>
