<script lang="ts">
    import { Button } from "$lib/components/ui/button/index.js";
    import ErrorView from "$lib/components/ErrorView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";
    import VectorView from "$lib/components/vector/VectorView.svelte";
    import { toVectorDeployments } from "$lib/deployments";
    import { getVectorDeployments } from "$lib/konfidence-api/queries.remote";
    import type { PageProps } from "./$types";

    const { data, params }: PageProps = $props();
    const vectorDeploymentsQuery = $derived(getVectorDeployments(params.projectId));
    const vectorDeployments = $derived.by(() => {
        if (vectorDeploymentsQuery.ready) {
            return toVectorDeployments(
                vectorDeploymentsQuery.current.landscapes,
                vectorDeploymentsQuery.current.stages,
                vectorDeploymentsQuery.current.vectorDeployments,
            );
        }
        return data.vectorDeployments;
    });
</script>

{#if vectorDeploymentsQuery.loading && !vectorDeployments}
    <LoadingView title="Loading vector deployments" message="Loading project vector deployments." />
{:else if vectorDeploymentsQuery.error && !vectorDeployments}
    <div class="grid justify-items-center gap-4 p-8">
        <ErrorView
            title="Failed to load vector deployments"
            message="Vector deployments are currently unavailable."
            error={vectorDeploymentsQuery.error}
        />
        <Button onclick={() => vectorDeploymentsQuery.refresh()}>Retry</Button>
    </div>
{:else if vectorDeployments}
    <VectorView {vectorDeployments} />
{/if}
