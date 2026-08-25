<script lang="ts">
    import "@ui5/webcomponents/dist/Label.js";
    import "@ui5/webcomponents/dist/Option.js";
    import "@ui5/webcomponents/dist/Select.js";

    import type { components } from "$lib/konfidence-api/schema";

    type Project = components["schemas"]["Project"];
    type SelectChangeEventDetail =
        import("@ui5/webcomponents/dist/Select.js").SelectChangeEventDetail;

    const { onselect, projects, selectedProjectId }: {
        onselect: (projectId: string) => void;
        projects: Project[];
        selectedProjectId?: string;
    } = $props();

    const handleChange = (event: CustomEvent<SelectChangeEventDetail>): void => {
        const projectId = event.detail.selectedOption.value;
        if (projectId) {
            onselect(projectId);
        }
    };
</script>

<div class="project-switcher">
    <ui5-label for="project-select">Project</ui5-label>
    <ui5-select
        id="project-select"
        accessible-name="Project"
        value={selectedProjectId ?? ""}
        onui5-change={handleChange}
    >
        <ui5-option value="">Select a project</ui5-option>
        {#each projects as project (project.id)}
            <ui5-option value={project.id}>{project.name}</ui5-option>
        {/each}
    </ui5-select>
</div>

<style>
    .project-switcher {
        display: grid;
        gap: 0.375rem;
        padding: 1rem;
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontSmallSize);
    }

    .project-switcher ui5-select {
        width: 100%;
        color: var(--sapTextColor);
        font: inherit;
    }
</style>
