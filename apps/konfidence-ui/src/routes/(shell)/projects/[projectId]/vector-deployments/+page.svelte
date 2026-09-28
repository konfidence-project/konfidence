<script lang="ts">
    import type { PageProps } from "./$types";
    import VectorDeploymentsView from "$lib/vector-deployments/VectorDeploymentsView.svelte";
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
    }));
    const vectorDeploymentsQuery = useVectorDeployments(() => ({
        landscapeId: data.landscapeId,
        projectId: data.projectId,
    }));

    const loading = $derived(
        landscapesQuery.loading ||
            stagesQuery.loading ||
            artifactDeploymentsQuery.loading ||
            vectorDeploymentsQuery.loading,
    );
    const error = $derived(
        vectorDeploymentsQuery.error ??
            landscapesQuery.error ??
            stagesQuery.error ??
            artifactDeploymentsQuery.error,
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

<svelte:head><title>Vector Deployments · Konfidence</title></svelte:head>

<VectorDeploymentsView
    projectId={data.projectId}
    artifactDeployments={artifactDeploymentsQuery.data ?? []}
    landscapes={landscapesQuery.data ?? []}
    stages={stagesQuery.data ?? []}
    vectorDeployments={vectorDeploymentsQuery.data ?? []}
    selectedLandscapeId={data.landscapeId}
    {loading}
    {hasLoaded}
    error={error?.message}
    onRetry={reload}
/>
