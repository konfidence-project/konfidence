<script lang="ts">
    import { page } from "$app/state";
    import ArtifactDeploymentsView from "$lib/artifact-deployments/ArtifactDeploymentsView.svelte";
    import { ArtifactDeploymentsStore } from "$lib/artifact-deployments/store.svelte";
    import { LANDSCAPE_PARAM, VECTOR_DEPLOYMENT_PARAM } from "$lib/artifact-deployments/params";
    import { getApiClient } from "$lib/konfidence-api/client-instance";

    const store = new ArtifactDeploymentsStore(getApiClient());

    const projectId = $derived(page.params.projectId as string);
    const landscapeId = $derived(page.url.searchParams.get(LANDSCAPE_PARAM) ?? undefined);
    const vectorDeploymentId = $derived(
        page.url.searchParams.get(VECTOR_DEPLOYMENT_PARAM) ?? undefined,
    );

    $effect(() => {
        void store.refresh(projectId, { landscapeId, vectorDeploymentId });
    });
</script>

<svelte:head><title>Artifact Deployments · Konfidence</title></svelte:head>

<ArtifactDeploymentsView
    rows={store.rows}
    landscapes={store.landscapes}
    vectorDeployments={store.vectorDeployments}
    selectedLandscapeId={landscapeId}
    selectedVectorDeploymentId={vectorDeploymentId}
    loading={store.status === "loading"}
    hasLoaded={store.hasLoaded}
    error={store.status === "error" ? store.error : undefined}
    onRetry={() => store.refresh(projectId, { landscapeId, vectorDeploymentId })}
/>
