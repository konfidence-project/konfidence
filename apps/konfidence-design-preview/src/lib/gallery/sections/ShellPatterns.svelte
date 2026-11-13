<script lang="ts">
    import {
        Avatar,
        AvatarGroup,
        Menu,
        ProjectSelection,
    } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let { name = "ProjectSelection" }: { name?: string } = $props();

    interface DemoProject {
        id: string;
        name: string;
    }

    const PROJECTS: readonly DemoProject[] = [
        { id: "konfidence", name: "Konfidence" },
        { id: "design-preview", name: "Design Preview" },
        { id: "aurora", name: "Aurora" },
        { id: "orbit", name: "Orbit" },
    ];

    let selectedProjectId = $state<string>(PROJECTS[0].id);
    let userMenuLog = $state<string>("");

    const selectedProject = $derived(
        PROJECTS.find((project) => project.id === selectedProjectId) ?? PROJECTS[0],
    );

    const handleProjectSelect = (details: { value: string }): void => {
        selectedProjectId = details.value;
    };

    const handleUserMenu = (details: { value: string }): void => {
        userMenuLog = `Selected: ${details.value}`;
    };
</script>

<Section
    id={name}
    title={name}
>
    <div class="stack flex flex-col gap-5">
        {#if name === "ProjectSelection"}
        <Sample
            title="Project switcher"
            description="Real <ProjectSelection> composed with <Menu>. Click to pick a project."
        >
            <Menu positioning={{ placement: "bottom-start" }} onSelect={handleProjectSelect}>
                <Menu.Trigger>
                    {#snippet element(attributes: Record<string, unknown>)}
                        <ProjectSelection
                            {...attributes}
                            name={selectedProject.name}
                        />
                    {/snippet}
                </Menu.Trigger>
                <Menu.Positioner>
                    <Menu.Content>
                        <Menu.ItemGroup>
                            <Menu.Label>Project</Menu.Label>
                            {#each PROJECTS as project (project.id)}
                                <Menu.Item
                                    value={project.id}
                                    active={project.id === selectedProjectId}
                                >
                                    {project.name}
                                </Menu.Item>
                            {/each}
                        </Menu.ItemGroup>
                    </Menu.Content>
                </Menu.Positioner>
            </Menu>
            <span class="log rounded-md bg-surface-subtle px-2 py-[3px] font-mono text-meta text-content-secondary">Active: <b>{selectedProject.name}</b></span>
        </Sample>

        {/if}
        {#if name === "Avatar"}
        <Sample title="Initials and online state">
            <Avatar initials="KO" ariaLabel="K. O'Malley" />
            <Avatar initials="MB" orbit ariaLabel="M. Barnes (online)" />
        </Sample>
        {/if}
        {#if name === "AvatarGroup"}
        <Sample title="Avatar group">
            <AvatarGroup>
                <Avatar initials="AA" />
                <Avatar initials="BB" />
                <Avatar initials="CC" />
                <Avatar initials="DD" />
            </AvatarGroup>
        </Sample>

        {/if}
        {#if name === "Menu"}
        <Sample
            title="User menu"
            previewHeight={480}
            description="Real <Menu.*> family — click the button to open. Selection is logged below."
        >
            <Menu positioning={{ placement: "bottom-end" }} onSelect={handleUserMenu}>
                <Menu.Trigger class="btn btn--secondary">Open user menu</Menu.Trigger>
                <Menu.Positioner>
                    <Menu.Content header>
                        <Menu.Header initials="KO">
                            {#snippet name()}Kaya O'Malley{/snippet}
                            {#snippet mail()}kaya@example.com{/snippet}
                        </Menu.Header>
                        <Menu.Separator />
                        <Menu.ItemGroup>
                            <Menu.Label>Account</Menu.Label>
                            <Menu.Item value="profile">Profile</Menu.Item>
                            <Menu.Item value="settings">Settings</Menu.Item>
                        </Menu.ItemGroup>
                        <Menu.Separator />
                        <Menu.Item value="signout" variant="danger">Sign out</Menu.Item>
                    </Menu.Content>
                </Menu.Positioner>
            </Menu>
            {#if userMenuLog}
                <span class="log rounded-md bg-surface-subtle px-2 py-[3px] font-mono text-meta text-content-secondary">{userMenuLog}</span>
            {/if}
        </Sample>

        {/if}
    </div>
</Section>
