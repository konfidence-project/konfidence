<script lang="ts">
    import { Label } from "$lib/components/ui/label/index.js";
    import * as NativeSelect from "$lib/components/ui/native-select/index.js";
    import type { components } from "$lib/konfidence-api/schema";

    type Project = components["schemas"]["ProjectResponse"];

    const { class: className, id, onselect, projects, selectedProjectId }: {
        class?: string;
        id: string;
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

<div class={["project-switcher grid gap-[0.4rem] border-b border-sidebar-border p-4", className]}>
    <Label for={id}>Project</Label>
    <NativeSelect.Root
        class="w-full"
        {id}
        aria-label="Project"
        value={selectedProjectId ?? ""}
        onchange={handleChange}
    >
        <NativeSelect.Option value="">Select a project</NativeSelect.Option>
        {#each projects as project (project.id)}
            <NativeSelect.Option value={project.id}>{project.name}</NativeSelect.Option>
        {/each}
    </NativeSelect.Root>
</div>
