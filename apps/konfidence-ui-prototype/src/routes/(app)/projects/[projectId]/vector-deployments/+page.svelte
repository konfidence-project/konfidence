<script lang="ts">
    import "@ui5/webcomponents/dist/Button.js";

    import ErrorView from "$lib/components/ErrorView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";
    import VectorView from "$lib/components/vector/VectorView.svelte";
    import { toVectorDeployments } from "$lib/deployments";
    import { getVectorDeployments } from "$lib/konfidence-api/queries.remote";
    import type { PageProps } from "./$types";

    const { params }: PageProps = $props();
    const vectorDeploymentsQuery = $derived(getVectorDeployments(params.projectId));
    const vectorDeployments = $derived.by(() => {
        if (!vectorDeploymentsQuery.ready) {
            return undefined;
        }
        return toVectorDeployments(
            vectorDeploymentsQuery.current.landscapes,
            vectorDeploymentsQuery.current.stages,
            vectorDeploymentsQuery.current.vectorDeployments,
        );
    });
</script>

{#if vectorDeploymentsQuery.loading && !vectorDeploymentsQuery.ready}
    <LoadingView title="Loading vector deployments" message="Loading project vector deployments." />
{:else if vectorDeploymentsQuery.error}
    <div class="query-error">
        <ErrorView
            title="Failed to load vector deployments"
            message="Vector deployments are currently unavailable."
            error={vectorDeploymentsQuery.error}
        />
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
        <ui5-button design="Emphasized" onclick={() => vectorDeploymentsQuery.refresh()}>Retry</ui5-button>
    </div>
{:else if vectorDeployments}
    <VectorView {vectorDeployments} />
{/if}

<style>
    .query-error {
        display: grid;
        justify-items: center;
        gap: 1rem;
        padding: 2rem;
    }
</style>
