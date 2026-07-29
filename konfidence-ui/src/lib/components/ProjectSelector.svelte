<script lang="ts">
    import GripIcon from "@lucide/svelte/icons/grip";
    import type { components } from "$lib/konfidence-api/schema";

    type Project = components["schemas"]["ProjectResponse"];

    let selectEl = $state<HTMLSelectElement>();

    const { collapsed = false, onexpand, onselect, projects, selectedProjectId }: {
        collapsed?: boolean;
        onexpand?: () => void;
        onselect: (projectId: string) => void;
        projects: Project[];
        selectedProjectId?: string;
    } = $props();

    const selectedProject = $derived(
        projects.find((p) => p.id === selectedProjectId),
    );

    const handleChange = (event: Event): void => {
        const projectId = (event.currentTarget as HTMLSelectElement).value;
        if (projectId) {
            onselect(projectId);
        }
    };

    export const openSelect = (): void => {
        selectEl?.showPicker();
    };
</script>

<div class={["project-switcher grid gap-1.5 border-b border-app-border p-4", collapsed && "min-[52rem]:min-h-14 min-[52rem]:gap-0 min-[52rem]:place-items-center min-[52rem]:p-0"]}>
    <label class={["text-[0.78rem] font-semibold text-app-muted", collapsed && "min-[52rem]:hidden"]} for="project-select">Project</label>
    <select bind:this={selectEl} id="project-select" class={["select min-h-[2.35rem] w-full border-app-border bg-app-card text-app-text", collapsed && "min-[52rem]:hidden"]} value={selectedProjectId ?? ""} onchange={handleChange}>
        <option value="">Select a project</option>
        {#each projects as project (project.id)}
            <option value={project.id}>{project.name}</option>
        {/each}
    </select>
    {#if collapsed}
        <div class="hidden min-[52rem]:block min-[52rem]:relative min-[52rem]:min-h-11 min-[52rem]:w-full">
            <button
                type="button"
                class="group flex min-h-11 w-full items-center gap-[0.7rem] rounded-[0.4rem] border-0 bg-transparent px-3 text-app-text no-underline hover:bg-app-accent/7 cursor-pointer [&_svg]:size-[1.1rem] [&_svg]:shrink-0 min-[52rem]:absolute min-[52rem]:inset-y-0 min-[52rem]:left-0 min-[52rem]:right-0 min-[52rem]:justify-center min-[52rem]:px-0 min-[52rem]:hover:right-auto min-[52rem]:hover:z-10 min-[52rem]:hover:w-max min-[52rem]:hover:pl-4 min-[52rem]:hover:pr-3 min-[52rem]:hover:bg-app-sidebar min-[52rem]:hover:shadow-md"
                aria-label="Open project selector"
                onclick={onexpand}
            >
                <GripIcon aria-hidden="true" />
                <span class="min-[52rem]:hidden min-[52rem]:group-hover:inline">Select a project</span>
            </button>
        </div>
    {/if}
</div>

<style>
    #project-select {
        appearance: none;
        padding: 0.5rem 2rem 0.5rem 0.5rem;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%23667781' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 0.5rem center;
        background-size: 1rem;
    }
</style>
