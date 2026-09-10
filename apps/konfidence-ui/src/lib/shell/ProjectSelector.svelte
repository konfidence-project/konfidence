<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { Menu, ProjectSwitcher } from "@konfidence/design-system/components";
    import { useProjects } from "$lib/projects/projects";
    import { isEmbedded } from "$lib/shell/embedded";

    /**
     * Project switcher trigger + popover menu. Composes the DS
     * `ProjectSwitcher` (renders `.project-switch` chrome) inside Skeleton's
     * `Menu.Trigger` via the `element` snippet, so Zag drives keyboard +
     * focus state on the real `<button>` `ProjectSwitcher` renders.
     */
    const projects = useProjects();

    const selected = $derived(
        projects.projects.find((project) => project.id === projects.selectedProjectId),
    );
    const displayName: string = $derived(selected?.name ?? "Select project");

    const handleSelect = (details: { value: string }): void => {
        const target = resolve("/(shell)/projects/[projectId]/landscape", {
            projectId: details.value,
        });
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- `target` is already a resolved pathname; we may append `?embedded=1` for host integrations.
        void goto(isEmbedded(page.url) ? `${target}?embedded=1` : target);
    };
</script>

<Menu positioning={{ placement: "bottom-start" }} onSelect={handleSelect}>
    <Menu.Trigger>
        {#snippet element(attributes)}
            <ProjectSwitcher {...attributes} name={displayName} data-testid="project-switch" />
        {/snippet}
    </Menu.Trigger>
    <Menu.Positioner>
        <Menu.Content>
            <Menu.ItemGroup>
                <Menu.Label>Project</Menu.Label>
                {#each projects.projects as project (project.id)}
                    <Menu.Item
                        value={project.id}
                        active={project.id === projects.selectedProjectId}
                        data-testid={`project-option-${project.id}`}
                    >
                        {project.name}
                    </Menu.Item>
                {/each}
            </Menu.ItemGroup>
        </Menu.Content>
    </Menu.Positioner>
</Menu>
