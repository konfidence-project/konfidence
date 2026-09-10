<script lang="ts">
    import AppShell from "../AppShell.svelte";
    import Avatar from "../../Avatar/Avatar.svelte";
    import NavGroup from "../../NavGroup/NavGroup.svelte";
    import NavItem from "../../NavItem/NavItem.svelte";
    import ProjectSwitcher from "../../ProjectSwitcher/ProjectSwitcher.svelte";
    import Sidebar from "../../Sidebar/Sidebar.svelte";
    import TopBar from "../../TopBar/TopBar.svelte";
</script>

<div data-testid="shell-root" style="min-height: 640px;">
    <AppShell>
        {#snippet topbar({ toggleDrawer })}
            <TopBar onHamburger={toggleDrawer}>
                {#snippet logo()}
                    <span
                        class="topbar__logo"
                        style="display: inline-flex; align-items: center; font-weight: var(--weight-bold); color: var(--text-primary);"
                        >Konfidence</span
                    >
                {/snippet}
                {#snippet switcher()}
                    <ProjectSwitcher name="Konfidence" aria-label="Change project" />
                {/snippet}
                {#snippet actions()}
                    <Avatar initials="AA" orbit ariaLabel="User menu" role="button" tabindex={0} />
                {/snippet}
            </TopBar>
        {/snippet}

        {#snippet sidebar({ closeDrawer })}
            <Sidebar>
                {#snippet mobileSwitcher()}
                    <ProjectSwitcher name="Konfidence" aria-label="Change project" />
                {/snippet}
                <NavGroup label="Delivery">
                    <NavItem
                        href="/landscape"
                        active
                        icon="grid"
                        onclick={closeDrawer}
                    >
                        Landscape
                    </NavItem>
                    <NavItem
                        href="/vectors"
                        icon="chain-link"
                        onclick={closeDrawer}
                    >
                        Vector Deployments
                    </NavItem>
                    <NavItem
                        href="/artifacts"
                        icon="product"
                        onclick={closeDrawer}
                    >
                        Artifact Deployments
                    </NavItem>
                </NavGroup>
            </Sidebar>
        {/snippet}

        {#snippet main()}
            <section style="padding: var(--space-4);">
                <h1
                    style="color: var(--text-primary); font-size: var(--text-h2); margin: 0;"
                >
                    Landscape
                </h1>
            </section>
        {/snippet}
    </AppShell>
</div>
