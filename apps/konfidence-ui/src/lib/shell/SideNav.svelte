<script lang="ts">
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { NavGroup, NavItem, Sidebar, isActive } from "@konfidence/design-system/components";
    import { useProjects } from "$lib/projects/projectContext";
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

    const destinations = $derived.by(() => {
        const projectId = projects.selectedProject?.id;
        if (!projectId) {
            return undefined;
        }
        return {
            artifactDeployments: resolve("/(shell)/projects/[projectId]/artifact-deployments", { projectId }),
            errorDemo: resolve("/(shell)/projects/[projectId]/error", { projectId }),
            landscape: resolve("/(shell)/projects/[projectId]/landscape", { projectId }),
            vectorDeployments: resolve("/(shell)/projects/[projectId]/vector-deployments", { projectId }),
        };
    });

    const activePath = $derived(page.url.pathname);
</script>

<Sidebar>
    {#snippet mobileSwitcher()}
        <ProjectSelector onSelect={closeDrawer} />
    {/snippet}

    {#if destinations}
    <NavGroup label="Delivery">
        <NavItem
            href={destinations.landscape}
            active={isActive(activePath, destinations.landscape)}
            icon="grid"
            data-testid="nav-landscape"
            onclick={closeDrawer}
        >
            Landscape
        </NavItem>
        <NavItem
            href={destinations.vectorDeployments}
            active={isActive(activePath, destinations.vectorDeployments)}
            icon="chain-link"
            data-testid="nav-vector-deployments"
            onclick={closeDrawer}
        >
            Vector Deployments
        </NavItem>
        <NavItem
            href={destinations.artifactDeployments}
            active={isActive(activePath, destinations.artifactDeployments)}
            icon="product"
            data-testid="nav-artifact-deployments"
            onclick={closeDrawer}
        >
            Artifact Deployments
        </NavItem>
    </NavGroup>
    <NavGroup label="Demo">
        <NavItem
            href={destinations.errorDemo}
            active={isActive(activePath, destinations.errorDemo)}
            icon="error"
            data-testid="nav-error"
            onclick={closeDrawer}
        >
            Error page
        </NavItem>
    </NavGroup>
    {/if}
</Sidebar>
