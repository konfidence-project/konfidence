<script lang="ts">
    import "@ui5/webcomponents/dist/Button.js";

    import ErrorView from "$lib/components/ErrorView.svelte";
    import LandscapeView from "$lib/components/LandscapeView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";
    import { getProjectLandscape } from "$lib/konfidence-api/queries.remote";
    import { toLandscapeView } from "$lib/stages";
    import type { PageProps } from "./$types";

    const { params }: PageProps = $props();
    const landscapeQuery = $derived(getProjectLandscape(params.projectId));
    const landscape = $derived.by(() => {
        if (!landscapeQuery.ready) {
            return undefined;
        }
        return toLandscapeView(
            landscapeQuery.current.landscapes,
            landscapeQuery.current.stages,
        );
    });
</script>

{#if landscapeQuery.loading && !landscapeQuery.ready}
    <LoadingView title="Loading landscape" message="Loading landscapes and stages." />
{:else if landscapeQuery.error}
    <div class="query-error">
        <ErrorView
            title="Failed to load landscape"
            message="The project landscape is currently unavailable."
            error={landscapeQuery.error}
        />
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
        <ui5-button design="Emphasized" onclick={() => landscapeQuery.refresh()}>Retry</ui5-button>
    </div>
{:else if landscape}
    <LandscapeView landscapes={landscape.landscapes} stages={landscape.stages} />
{/if}

<style>
    :global(.content) {
        padding: 0;
    }

    .query-error {
        display: grid;
        justify-items: center;
        gap: 1rem;
        padding: 2rem;
    }
</style>
