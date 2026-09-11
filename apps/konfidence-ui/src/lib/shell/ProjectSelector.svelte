<script lang="ts">
    import { Menu, ProjectSelection } from "@konfidence/design-system/components";
    import { useProjects } from "$lib/projects/projectContext";

    /**
     * Project switcher trigger + popover menu. Skeleton's `Menu.Trigger`
     * supplies the Zag attributes for keyboard, focus, and accessibility.
     */
    const projects = useProjects();

    interface Props { onSelect?: () => void; }
    let { onSelect }: Props = $props();

    const displayName: string = $derived(projects.selectedProject?.name ?? "Select project");

    const handleSelect = (details: { value: string }): void => {
        onSelect?.();
        void projects.selectProject(details.value);
    };
</script>

{#if projects.status === "ready"}
    <Menu positioning={{ placement: "bottom-start" }} onSelect={handleSelect}>
        <Menu.Trigger>
            {#snippet element(attributes)}
                <ProjectSelection
                    {...attributes}
                    name={displayName}
                    class={attributes.class}
                    data-testid="project-switch"
                />
            {/snippet}
        </Menu.Trigger>
        <Menu.Positioner>
            <Menu.Content>
                <Menu.ItemGroup>
                    <Menu.Label>Project</Menu.Label>
                    {#each projects.projects as project (project.id)}
                        <Menu.Item
                            value={project.id}
                            active={project.id === projects.selectedProject?.id}
                            data-testid={`project-option-${project.id}`}
                        >
                            {project.name}
                        </Menu.Item>
                    {/each}
                </Menu.ItemGroup>
            </Menu.Content>
        </Menu.Positioner>
    </Menu>
{/if}
