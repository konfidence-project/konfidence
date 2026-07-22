<script lang="ts">
    import "@ui5/webcomponents-fiori/dist/Page.js";

    import ErrorView from "$lib/components/ErrorView.svelte";
    import LandscapeView from "$lib/components/LandscapeView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";

    import { getStages } from "./stages.remote";

    const stagesQuery = $derived(getStages());
</script>

{#if stagesQuery.loading && !stagesQuery.ready}
    <LoadingView
        title="Loading stages"
        message="Fetching mock Stage resources and their latest conditions."
    />
{:else if stagesQuery.error}
    <ErrorView
        title="Failed to load stages"
        message="The stages overview is currently unavailable."
        error={stagesQuery.error}
    />
{:else if stagesQuery.ready}
    {#if stagesQuery.current.accessError}
        <ErrorView
            title={stagesQuery.current.accessError.title}
            message={stagesQuery.current.accessError.message}
            status={stagesQuery.current.accessError.status}
        />
    {:else}
        <section class="stage-landscape" aria-label="Stage promotion landscape">
            <LandscapeView stages={stagesQuery.current.items} />
        </section>
    {/if}
{/if}

<style>
    :global(.content) {
        padding: 0;
    }

    .stage-landscape {
        display: flex;
        flex: 1;
        width: 100%;
        min-height: 0;
        height: 100%;
        box-sizing: border-box;
    }
</style>
