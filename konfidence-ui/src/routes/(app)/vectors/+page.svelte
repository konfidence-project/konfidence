<script lang="ts">
    import "@ui5/webcomponents-fiori/dist/Page.js";

    import ErrorView from "$lib/components/ErrorView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";
    import VectorView from "$lib/components/vector/VectorView.svelte";

    import { getVectors } from "./vectors.remote";

    // Number of mock vectors to request. Bump this up to stress-test the
    // Table's virtualization and growing with large data sets.
    const VECTOR_COUNT = 10_000;

    const vectorsQuery = $derived(getVectors(VECTOR_COUNT));
</script>

{#if vectorsQuery.loading && !vectorsQuery.ready}
    <LoadingView
        title="Loading vectors"
        message="Fetching mock Vector resources and their latest conditions."
    />
{:else if vectorsQuery.error}
    <ErrorView
        title="Failed to load vectors"
        message="The vectors overview is currently unavailable."
        error={vectorsQuery.error}
    />
{:else if vectorsQuery.ready}
    <section class="vector-view" aria-label="Vector Overview">
        <VectorView vectors={vectorsQuery.current.items} />
    </section>
{/if}

<style>
    :global(.content) {
        padding: 0;
    }

    .vector-view {
        display: flex;
        flex: 1;
        width: 100%;
        height: 100%;
        box-sizing: border-box;
    }
</style>
