<script lang="ts">
    import type { PageProps } from "./$types";
    import VectorDeploymentsView from "$lib/vector-deployments/VectorDeploymentsView.svelte";

    let { data }: PageProps = $props();
</script>

<svelte:head><title>Vector Deployments · Konfidence</title></svelte:head>

<VectorDeploymentsView
    projectId={data.projectId}
    artifactDeployments={data.store.artifactDeployments}
    landscapes={data.store.landscapes}
    stages={data.store.stages}
    vectorDeployments={data.store.vectorDeployments}
    selectedLandscapeId={data.landscapeId}
    loading={data.store.status === "loading"}
    hasLoaded={data.store.hasLoaded}
    error={data.store.status === "error" ? data.store.error : undefined}
    onRetry={() =>
        data.store.refresh(data.projectId, {
            landscapeId: data.landscapeId,
        })}
/>
