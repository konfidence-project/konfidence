<script lang="ts">
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { useProjects } from "$lib/projects/projects";
    import ProjectSelector from "$lib/shell/ProjectSelector.svelte";
    import { isActive } from "$lib/shell/nav-destinations";
    import { closeDrawer } from "$lib/shell/sidebar.svelte";

    /**
     * Primary side navigation. The three delivery destinations are unrolled
     * statically so each `href` resolves through `$app/paths`;
     * `aria-current="page"` follows the URL, and clicking a link closes the
     * mobile drawer. A "Demo" group carries throwaway links used to eyeball
     * cross-cutting behaviour like the error boundary.
     */
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

<nav class="sidebar" aria-label="Primary">
    <div class="sidebar__project">
        <ProjectSelector />
    </div>
    <div class="nav-group">
        <div class="nav-group__label">Delivery</div>
        <a
            class="nav-item"
            class:nav-item--active={isActive(activePath, landscape)}
            href={landscape}
            aria-current={isActive(activePath, landscape) ? "page" : undefined}
            data-testid="nav-landscape"
            onclick={closeDrawer}
        >
            <span>Landscape</span>
        </a>
        <a
            class="nav-item"
            class:nav-item--active={isActive(activePath, vectorDeployments)}
            href={vectorDeployments}
            aria-current={isActive(activePath, vectorDeployments) ? "page" : undefined}
            data-testid="nav-vector-deployments"
            onclick={closeDrawer}
        >
            <span>Vector Deployments</span>
        </a>
        <a
            class="nav-item"
            class:nav-item--active={isActive(activePath, artifactDeployments)}
            href={artifactDeployments}
            aria-current={isActive(activePath, artifactDeployments) ? "page" : undefined}
            data-testid="nav-artifact-deployments"
            onclick={closeDrawer}
        >
            <span>Artifact Deployments</span>
        </a>
    </div>
    <div class="nav-group">
        <div class="nav-group__label">Demo</div>
        <a
            class="nav-item"
            class:nav-item--active={isActive(activePath, errorDemo)}
            href={errorDemo}
            aria-current={isActive(activePath, errorDemo) ? "page" : undefined}
            data-testid="nav-error"
            onclick={closeDrawer}
        >
            <span>Error page</span>
        </a>
    </div>
</nav>
