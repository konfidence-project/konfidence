<script lang="ts">
    import {
        AppShell,
        Menu,
        NavGroup,
        NavItem,
        ProjectSelection,
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
        /**
         * SAP-icons v5 name (rendered by the underlying `<ui5-icon>`).
         * Matches the current `konfidence-ui` shell vocabulary.
         */
        icon: "grid" | "chain-link" | "product" | "error" | "log";
        group: "delivery" | "insight";
    }

    const PROJECTS: readonly DemoProject[] = [
        { id: "konfidence", name: "Konfidence" },
        { id: "design-preview", name: "Design Preview" },
        { id: "aurora", name: "Aurora" },
        { id: "orbit", name: "Orbit" },
    ];

    const NAV: readonly DemoNavKey[] = [
        { group: "delivery", icon: "grid", id: "landscape", label: "Landscape" },
        { group: "delivery", icon: "chain-link", id: "vectors", label: "Vector Deployments" },
        { group: "delivery", icon: "product", id: "artifacts", label: "Artifact Deployments" },
        { group: "insight", icon: "error", id: "analytics", label: "Analytics" },
        { group: "insight", icon: "log", id: "activities", label: "Activities" },
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
    <div class="shell-frame min-h-[480px] max-h-[640px] resize-y overflow-hidden rounded-xl border border-outline-subtle">
        <AppShell>
            {#snippet topbar({ toggleDrawer }: { toggleDrawer: () => void })}
                <TopBar onHamburger={toggleDrawer}>
                    {#snippet logo()}
                        <span class="shell-logo inline-flex items-center font-bold tracking-[-0.02em] text-content-primary">Konfidence</span>
                    {/snippet}
                    {#snippet switcher()}
                        <Menu
                            positioning={{ placement: "bottom-start" }}
                            onSelect={handleProjectSelect}
                        >
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
                        <ProjectSelection name={selectedProject.name} />
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
                <section class="shell-main px-7 py-8">
                    <p class="shell-main__eyebrow mb-1 text-meta font-semibold tracking-[0.06em] text-content-tertiary uppercase">
                        {selectedProject.name} · {selectedNav.label}
                    </p>
                    <h1 class="shell-main__title mb-2 text-h1 font-display tracking-[-0.5px] text-content-primary">{selectedNav.label}</h1>
                    <p class="shell-main__lead m-0 max-w-[640px] text-compact text-content-secondary">
                        Live application shell — real routing intent, real drawer behavior, real
                        design-system components. Pick a project in the topbar switcher, tap a
                        nav item, or open the user menu on the right.
                    </p>
                    {#if userMenuLog}
                        <p class="shell-main__log mt-3 rounded-lg border border-dashed border-outline-subtle bg-surface-subtle px-3 py-2 font-mono text-meta text-content-primary">{userMenuLog}</p>
                    {/if}
                </section>
            {/snippet}
        </AppShell>
    </div>
</Section>
