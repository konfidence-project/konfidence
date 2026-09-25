<script lang="ts">
    import type { PageProps } from "./$types";
    import VectorDeploymentsView from "$lib/vector-deployments/VectorDeploymentsView.svelte";
    import {
        useArtifactDeployments,
        useLandscapes,
        useStages,
        useVectorDeployments,
    } from "$lib/deployments/queries.svelte.js";

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

    // Vector deployments is the primary resource for this view: its error drives
    // the page-level error state.
    const loading = $derived(vectorDeploymentsQuery.loading);
    const error = $derived(vectorDeploymentsQuery.error);
    const hasLoaded = $derived(vectorDeploymentsQuery.data !== undefined);

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
    {error}
    onRetry={reload}
/>
