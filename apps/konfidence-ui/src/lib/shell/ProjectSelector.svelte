<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { Menu } from "@skeletonlabs/skeleton-svelte";
    import { useProjects } from "$lib/projects/projects";
    import { isEmbedded } from "$lib/shell/embedded";

    /**
     * Project switcher trigger + popover menu. Selecting a project navigates
     * to that project's landscape. Nothing persists client-side yet — the
     * URL is the state.
     */
    const projects = useProjects();

    const selected = $derived(
        projects.projects.find((project) => project.id === projects.selectedProjectId),
    );

    const handleSelect = (details: { value: string }): void => {
        const target = resolve("/(shell)/projects/[projectId]/landscape", {
            projectId: details.value,
        });
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- `target` is already a resolved pathname; we may append `?embedded=1` for host integrations.
        void goto(isEmbedded(page.url) ? `${target}?embedded=1` : target);
    };
</script>

<Menu positioning={{ placement: "bottom-start" }} onSelect={handleSelect}>
    <Menu.Trigger class="proj-switch" data-testid="project-switch">
        <span class="proj-switch__name">{selected?.name ?? "Select project"}</span>
        <span aria-hidden="true">▾</span>
    </Menu.Trigger>
    <Menu.Positioner class="menu-positioner">
        <Menu.Content class="menu">
            <Menu.ItemGroup>
                <Menu.ItemGroupLabel class="menu__label">Project</Menu.ItemGroupLabel>
                {#each projects.projects as project (project.id)}
                    <Menu.Item
                        value={project.id}
                        class="menu__item {project.id === projects.selectedProjectId
                            ? 'menu__item--active'
                            : ''}"
                        data-testid={`project-option-${project.id}`}
                    >
                        <span class="menu__text">{project.name}</span>
                    </Menu.Item>
                {/each}
            </Menu.ItemGroup>
        </Menu.Content>
    </Menu.Positioner>
</Menu>
