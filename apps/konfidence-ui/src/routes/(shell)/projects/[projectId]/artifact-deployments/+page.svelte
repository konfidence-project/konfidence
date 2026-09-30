<script lang="ts">
    import ArtifactDeploymentsView from "$lib/artifact-deployments/ArtifactDeploymentsView.svelte";
    import type { PageProps } from "./$types";
    import {
        useArtifactDeployments,
        useLandscapes,
        useStages,
        useVectorDeployments,
    } from "$lib/queries.svelte.js";

    let { data }: PageProps = $props();

    const landscapesQuery = useLandscapes(() => ({ projectId: data.projectId }));
    const stagesQuery = useStages(() => ({ projectId: data.projectId }));
    const artifactDeploymentsQuery = useArtifactDeployments(() => ({
        landscapeId: data.landscapeId,
        projectId: data.projectId,
        vectorDeploymentId: data.vectorDeploymentId,
    }));
    // The vector-deployment dropdown needs the full, unfiltered list, so this
    // query is invoked without a landscape filter (matching the previous store).
    const vectorDeploymentsQuery = useVectorDeployments(() => ({ projectId: data.projectId }));

    const loading = $derived(
        landscapesQuery.loading ||
            stagesQuery.loading ||
            artifactDeploymentsQuery.loading ||
            vectorDeploymentsQuery.loading,
    );
    const error = $derived(
        artifactDeploymentsQuery.error ??
            landscapesQuery.error ??
            stagesQuery.error ??
            vectorDeploymentsQuery.error,
    );
    const hasLoaded = $derived(
        landscapesQuery.data !== undefined &&
            stagesQuery.data !== undefined &&
            artifactDeploymentsQuery.data !== undefined &&
            vectorDeploymentsQuery.data !== undefined,
    );

    const reload = (): void => {
        landscapesQuery.reload();
        stagesQuery.reload();
        artifactDeploymentsQuery.reload();
        vectorDeploymentsQuery.reload();
    };
</script>

<svelte:head><title>Artifact Deployments · Konfidence</title></svelte:head>

<ArtifactDeploymentsView
    artifactDeployments={artifactDeploymentsQuery.data ?? []}
    landscapes={landscapesQuery.data ?? []}
    stages={stagesQuery.data ?? []}
    vectorDeployments={vectorDeploymentsQuery.data ?? []}
    selectedLandscapeId={data.landscapeId}
    selectedVectorDeploymentId={data.vectorDeploymentId}
    {loading}
    {hasLoaded}
    error={error?.message}
    onRetry={reload}
/>
