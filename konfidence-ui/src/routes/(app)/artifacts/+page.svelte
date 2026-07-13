<script lang="ts">
    import ArtifactsView from "$lib/components/artifacts/ArtifactsView.svelte";
    import ErrorView from "$lib/components/ErrorView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";

    import { getArtifactsData } from "./artifacts.remote";

    const artifactsQuery = $derived(getArtifactsData());
</script>

{#if artifactsQuery.loading && !artifactsQuery.ready}
    <LoadingView
        title="Loading artifacts"
        message="Fetching artifact deployments."
    />
{:else if artifactsQuery.error}
    <ErrorView
        title="Failed to load artifacts"
        message="The artifacts dashboard is currently unavailable."
        error={artifactsQuery.error}
    />
{:else if artifactsQuery.ready}
    <ArtifactsView artifacts={artifactsQuery.current.items} />
{/if}
