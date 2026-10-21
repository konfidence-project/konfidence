<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents-icons/dist/question-mark.js";
    import { Button, EmptyState, OrbitLoader } from "@konfidence/design-system/components";
    import { HTTP_NOT_FOUND } from "#lib/http-status.ts";
    import { useVectorDeployment } from "#lib/queries.svelte.ts";
    import VectorDeploymentFields from "#lib/vector-deployments/VectorDeploymentFields.svelte";

    interface Props {
        projectId: string;
        landscapeId: string;
        vectorDeploymentId: string;
    }
    let { projectId, landscapeId, vectorDeploymentId }: Props = $props();

    const query = useVectorDeployment(() => ({ landscapeId, projectId, vectorDeploymentId }));
    const notFound = $derived(query.error?.status === HTTP_NOT_FOUND);
</script>

{#if query.loading}
    <OrbitLoader label="Loading vector deployment" />
{:else if query.error && !notFound}
    <EmptyState
        tone="error"
        title="Vector deployment could not be loaded"
        description={query.error.message}
    >
        {#snippet icon()}<ui5-icon name="error" aria-hidden="true"></ui5-icon>{/snippet}
        {#snippet action()}
            <Button variant="secondary" onclick={() => query.reload()}
                ><span>Retry</span></Button
            >
        {/snippet}
    </EmptyState>
{:else if notFound || !query.data}
    <EmptyState
        tone="empty"
        title="No vector deployment"
        description="This stage version has no vector deployment."
    >
        {#snippet icon()}<ui5-icon name="question-mark" aria-hidden="true"
            ></ui5-icon>{/snippet}
    </EmptyState>
{:else}
    <VectorDeploymentFields vectorDeployment={query.data} />
{/if}
