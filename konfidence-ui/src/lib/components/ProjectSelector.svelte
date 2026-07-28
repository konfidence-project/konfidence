<script lang="ts">
    import type { components } from "$lib/konfidence-api/schema";

    type Project = components["schemas"]["ProjectResponse"];

    const { collapsed = false, onselect, projects, selectedProjectId }: {
        collapsed?: boolean;
        onselect: (projectId: string) => void;
        projects: Project[];
        selectedProjectId?: string;
    } = $props();

    const handleChange = (event: Event): void => {
        const projectId = (event.currentTarget as HTMLSelectElement).value;
        if (projectId) {
            onselect(projectId);
        }
    };
</script>

<div class={["project-switcher grid gap-1.5 border-b border-app-border p-4", collapsed && "min-[52rem]:min-h-14 min-[52rem]:gap-0 min-[52rem]:p-0"]}>
    <label class={["text-[0.78rem] font-semibold text-app-muted", collapsed && "min-[52rem]:hidden"]} for="project-select">Project</label>
    <select id="project-select" class={["select min-h-[2.35rem] w-full border-app-border bg-app-card text-app-text", collapsed && "min-[52rem]:hidden"]} value={selectedProjectId ?? ""} onchange={handleChange}>
        <option value="">Select a project</option>
        {#each projects as project (project.id)}
            <option value={project.id}>{project.name}</option>
        {/each}
    </select>
</div>
