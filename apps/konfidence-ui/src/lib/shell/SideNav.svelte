<script lang="ts">
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { NavGroup, NavItem, Sidebar, isActive } from "@konfidence/design-system/components";
    import { useProjects } from "$lib/projects/projects";
    import ProjectSelector from "$lib/shell/ProjectSelector.svelte";

    /**
     * Primary side navigation. Composes the design-system `Sidebar` /
     * `NavGroup` / `NavItem` primitives; the app owns the destination list,
     * `$app/paths.resolve` calls, and the active-state derivation from the
     * router. Below the `md` breakpoint the project switcher moves from the
     * topbar into the sidebar drawer via the `mobileSwitcher` snippet.
     *
     * The `closeDrawer` prop is passed down by the shell layout so each
     * destination click collapses the mobile drawer.
     */
    interface Props {
        closeDrawer: () => void;
    }

    let { closeDrawer }: Props = $props();

    const projects = useProjects();

    const landscape = $derived(
        resolve("/(shell)/projects/[projectId]/landscape", {
            projectId: projects.selectedProjectId,
        }),
    );
    const vectorDeployments = $derived(
        resolve("/(shell)/projects/[projectId]/vector-deployments", {
            projectId: projects.selectedProjectId,
        }),
    );
    const artifactDeployments = $derived(
        resolve("/(shell)/projects/[projectId]/artifact-deployments", {
            projectId: projects.selectedProjectId,
        }),
    );
    const errorDemo = $derived(
        resolve("/(shell)/projects/[projectId]/error", {
            projectId: projects.selectedProjectId,
        }),
    );

    const activePath = $derived(page.url.pathname);
</script>

<Sidebar>
    {#snippet mobileSwitcher()}
        <ProjectSelector />
    {/snippet}

    <NavGroup label="Delivery">
        <NavItem
            href={landscape}
            active={isActive(activePath, landscape)}
            icon="grid"
            data-testid="nav-landscape"
            onclick={closeDrawer}
        >
            Landscape
        </NavItem>
        <NavItem
            href={vectorDeployments}
            active={isActive(activePath, vectorDeployments)}
            icon="chain-link"
            data-testid="nav-vector-deployments"
            onclick={closeDrawer}
        >
            Vector Deployments
        </NavItem>
        <NavItem
            href={artifactDeployments}
            active={isActive(activePath, artifactDeployments)}
            icon="product"
            data-testid="nav-artifact-deployments"
            onclick={closeDrawer}
        >
            Artifact Deployments
        </NavItem>
    </NavGroup>
    <NavGroup label="Demo">
        <NavItem
            href={errorDemo}
            active={isActive(activePath, errorDemo)}
            icon="error"
            data-testid="nav-error"
            onclick={closeDrawer}
        >
            Error page
        </NavItem>
    </NavGroup>
</Sidebar>
