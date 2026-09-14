<script lang="ts">
    import ArtifactDeploymentsView from "$lib/artifact-deployments/ArtifactDeploymentsView.svelte";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();
</script>

<svelte:head><title>Artifact Deployments · Konfidence</title></svelte:head>

<ArtifactDeploymentsView
    artifactDeployments={data.store.artifactDeployments}
    landscapes={data.store.landscapes}
    stages={data.store.stages}
    vectorDeployments={data.store.vectorDeployments}
    selectedLandscapeId={data.landscapeId}
    selectedVectorDeploymentId={data.vectorDeploymentId}
    loading={data.store.status === "loading"}
    hasLoaded={data.store.hasLoaded}
    error={data.store.status === "error" ? data.store.error : undefined}
    onRetry={() =>
        data.store.refresh(data.projectId, {
            landscapeId: data.landscapeId,
            vectorDeploymentId: data.vectorDeploymentId,
        })}
/>
