<script lang="ts">
    import {
        Avatar,
        AvatarGroup,
        Menu,
        ProjectSwitcher,
    } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

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
    id="shell-patterns"
    title="Shell patterns"
    subtitle="Command palette, tabs-with-overflow, status bar — plus the real components already shipped for user menu, avatar groups, and project switcher. All interactive with dummy data."
    status="partial"
>
    <div class="stack">
        <Sample
            title="Project switcher"
            description="Real <ProjectSwitcher> composed with <Menu>. Click to pick a project."
        >
            <Menu positioning={{ placement: "bottom-start" }} onSelect={handleProjectSelect}>
                <Menu.Trigger>
                    {#snippet element(attributes: Record<string, unknown>)}
                        <ProjectSwitcher
                            {...attributes}
                            name={selectedProject.name}
                            aria-label="Change project"
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
            <span class="log">Active: <b>{selectedProject.name}</b></span>
        </Sample>

        <Sample title="Avatar & AvatarGroup">
            <Avatar initials="KO" ariaLabel="K. O'Malley" />
            <Avatar initials="MB" orbit ariaLabel="M. Barnes (online)" />
            <AvatarGroup>
                <Avatar initials="AA" />
                <Avatar initials="BB" />
                <Avatar initials="CC" />
                <Avatar initials="DD" />
            </AvatarGroup>
        </Sample>

        <Sample
            title="User menu"
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
                <span class="log">{userMenuLog}</span>
            {/if}
        </Sample>

        <NotYetImplemented
            note="Additional shell patterns from the reference design are still pending."
            planned={[
                "Tabs with overflow menu",
                "Command palette / global search",
                "Status bar (footer chrome)",
                "In-context help tour",
            ]}
        />
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .log {
        font-family: var(--font-mono);
        font-size: var(--text-meta);
        color: var(--text-secondary);
        padding: 3px 8px;
        border-radius: 6px;
        background: var(--surface-subtle);
    }
</style>
