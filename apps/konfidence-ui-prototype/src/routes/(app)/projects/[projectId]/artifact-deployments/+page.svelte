<script lang="ts">
    import "@ui5/webcomponents/dist/Button.js";

    import ArtifactsView from "$lib/components/artifacts/ArtifactsView.svelte";
    import ErrorView from "$lib/components/ErrorView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";
    import { toArtifactDeployments } from "$lib/deployments";
    import { getArtifactDeployments } from "$lib/konfidence-api/queries.svelte";
    import type { PageProps } from "./$types";

    const { params }: PageProps = $props();
    const deploymentsQuery = $derived(getArtifactDeployments(params.projectId));
    const deployments = $derived.by(() => {
        const current = deploymentsQuery.current;
        if (!current) {
            return undefined;
        }
        return toArtifactDeployments(
            current.landscapes,
            current.stages,
            current.artifactDeployments,
        );
    });
</script>

{#if deploymentsQuery.loading && !deploymentsQuery.ready}
    <LoadingView title="Loading artifact deployments" message="Loading project artifact deployments." />
{:else if deploymentsQuery.error}
    <div class="query-error">
        <ErrorView
            title="Failed to load artifact deployments"
            message="Artifact deployments are currently unavailable."
            error={deploymentsQuery.error}
        />
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
        <ui5-button design="Emphasized" onclick={() => deploymentsQuery.refresh()}>Retry</ui5-button>
    </div>
{:else if deployments}
    <ArtifactsView {deployments} />
{/if}

<style>
    .query-error {
        display: grid;
        justify-items: center;
        gap: 1rem;
        padding: 2rem;
    }
</style>
