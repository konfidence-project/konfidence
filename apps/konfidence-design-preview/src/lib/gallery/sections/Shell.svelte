<script lang="ts">
    import {
        AppShell,
        Avatar,
        Menu,
        NavGroup,
        NavItem,
        ProjectSwitcher,
        Sidebar,
        TopBar,
    } from "@konfidence/design-system/components";
    import Section from "../Section.svelte";

    interface DemoProject {
        id: string;
        name: string;
    }

    interface DemoNavKey {
        id: string;
        label: string;
        icon:
            | "landscape"
            | "vectors"
            | "artifacts"
            | "analytics"
            | "activities";
        group: "delivery" | "insight";
    }

    const PROJECTS: readonly DemoProject[] = [
        { id: "konfidence", name: "Konfidence" },
        { id: "design-preview", name: "Design Preview" },
        { id: "aurora", name: "Aurora" },
        { id: "orbit", name: "Orbit" },
    ];

    const NAV: readonly DemoNavKey[] = [
        { group: "delivery", icon: "landscape", id: "landscape", label: "Landscape" },
        { group: "delivery", icon: "vectors", id: "vectors", label: "Vector Deployments" },
        { group: "delivery", icon: "artifacts", id: "artifacts", label: "Artifact Deployments" },
        { group: "insight", icon: "analytics", id: "analytics", label: "Analytics" },
        { group: "insight", icon: "activities", id: "activities", label: "Activities" },
    ];

    let selectedProjectId = $state<string>(PROJECTS[0].id);
    let selectedNavId = $state<string>("landscape");
    let userMenuLog = $state<string>("");

    const selectedProject = $derived(
        PROJECTS.find((project) => project.id === selectedProjectId) ?? PROJECTS[0],
    );
    const selectedNav = $derived(NAV.find((item) => item.id === selectedNavId) ?? NAV[0]);

    const deliveryItems = $derived(NAV.filter((item) => item.group === "delivery"));
    const insightItems = $derived(NAV.filter((item) => item.group === "insight"));

    const handleProjectSelect = (details: { value: string }): void => {
        selectedProjectId = details.value;
    };

    const handleUserMenu = (details: { value: string }): void => {
        userMenuLog = `Selected: ${details.value}`;
    };
</script>

<Section
    id="shell"
    title="Application shell"
    subtitle="AppShell + TopBar + Sidebar (from #893). Project switcher, user menu, and sidebar navigation are wired up with dummy data — try clicking around."
    status="implemented"
>
    <div class="shell-frame">
        <AppShell>
            {#snippet topbar({ toggleDrawer }: { toggleDrawer: () => void })}
                <TopBar onHamburger={toggleDrawer}>
                    {#snippet logo()}
                        <span class="shell-logo">Konfidence</span>
                    {/snippet}
                    {#snippet switcher()}
                        <Menu
                            positioning={{ placement: "bottom-start" }}
                            onSelect={handleProjectSelect}
                        >
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
                    {/snippet}
                    {#snippet actions()}
                        <Menu
                            positioning={{ placement: "bottom-end" }}
                            onSelect={handleUserMenu}
                        >
                            <Menu.Trigger
                                class="avatar avatar--orbit"
                                aria-label="Open user menu"
                            >
                                KO
                            </Menu.Trigger>
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
                                    <Menu.Item value="signout" variant="danger">
                                        Sign out
                                    </Menu.Item>
                                </Menu.Content>
                            </Menu.Positioner>
                        </Menu>
                    {/snippet}
                </TopBar>
            {/snippet}

            {#snippet sidebar({ closeDrawer }: { closeDrawer: () => void })}
                <Sidebar>
                    {#snippet mobileSwitcher()}
                        <ProjectSwitcher
                            name={selectedProject.name}
                            aria-label="Change project"
                        />
                    {/snippet}
                    <NavGroup label="Delivery">
                        {#each deliveryItems as item (item.id)}
                            <NavItem
                                href="#shell"
                                active={item.id === selectedNavId}
                                icon={item.icon}
                                onclick={() => {
                                    selectedNavId = item.id;
                                    closeDrawer();
                                }}
                            >
                                {item.label}
                            </NavItem>
                        {/each}
                    </NavGroup>
                    <NavGroup label="Insight">
                        {#each insightItems as item (item.id)}
                            <NavItem
                                href="#shell"
                                active={item.id === selectedNavId}
                                icon={item.icon}
                                onclick={() => {
                                    selectedNavId = item.id;
                                    closeDrawer();
                                }}
                            >
                                {item.label}
                            </NavItem>
                        {/each}
                    </NavGroup>
                </Sidebar>
            {/snippet}

            {#snippet main()}
                <section class="shell-main">
                    <p class="shell-main__eyebrow">
                        {selectedProject.name} · {selectedNav.label}
                    </p>
                    <h1 class="shell-main__title">{selectedNav.label}</h1>
                    <p class="shell-main__lead">
                        Live application shell — real routing intent, real drawer behavior, real
                        design-system components. Pick a project in the topbar switcher, tap a
                        nav item, or open the user menu on the right.
                    </p>
                    {#if userMenuLog}
                        <p class="shell-main__log">{userMenuLog}</p>
                    {/if}
                </section>
            {/snippet}
        </AppShell>
    </div>
</Section>

<style>
    .shell-frame {
        border: 1px solid var(--border-subtle);
        border-radius: 12px;
        overflow: hidden;
        min-height: 480px;
        max-height: 640px;
        resize: vertical;
    }

    .shell-logo {
        display: inline-flex;
        align-items: center;
        font-weight: var(--weight-bold, 700);
        color: var(--text-primary);
        letter-spacing: -0.02em;
    }

    .shell-main {
        padding: 32px 28px;
    }

    .shell-main__eyebrow {
        margin: 0 0 4px;
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        text-transform: uppercase;
        letter-spacing: 0.06em;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .shell-main__title {
        margin: 0 0 8px;
        font-size: var(--text-h1);
        font-weight: var(--weight-display, 600);
        color: var(--text-primary);
        letter-spacing: -0.5px;
    }

    .shell-main__lead {
        margin: 0;
        color: var(--text-secondary);
        max-width: 640px;
        font-size: var(--text-sm);
    }

    .shell-main__log {
        margin: 12px 0 0;
        padding: 8px 12px;
        border: 1px dashed var(--border-subtle);
        border-radius: 8px;
        background: var(--surface-subtle);
        color: var(--text-primary);
        font-family: var(--font-mono);
        font-size: var(--text-meta);
    }
</style>
