<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents-icons/dist/question-mark.js";
    import { Button, EmptyState, OrbitLoader } from "@konfidence/design-system/components";
    import type { Stage } from "#lib/konfidence-api/types.ts";
    import StageVersionDetails from "#lib/landscape/components/StageVersionDetails.svelte";
    import VectorDeploymentSection from "#lib/landscape/components/VectorDeploymentSection.svelte";

    interface Props {
        error?: string;
        errorStatus?: number;
        projectId?: string;
        stage?: Stage;
        status: "loading" | "ready" | "error";
        onRetry: () => void;
    }
    let { error, errorStatus, projectId, stage, status, onRetry }: Props = $props();

    const targetVersionId = $derived(stage?.targetStageVersion?.id);
    const activeVersionId = $derived(stage?.activeStageVersion?.id);
    const collapsed = $derived(
        targetVersionId !== undefined && targetVersionId === activeVersionId,
    );
</script>
{#if status === "loading"}
    <OrbitLoader label="Loading stage details" />
{:else if status === "error" && errorStatus !== 404}
    <EmptyState
        tone="error"
        title="Stage details could not be loaded"
        description={error}
    >
        {#snippet icon()}<ui5-icon name="error" aria-hidden="true"></ui5-icon>{/snippet}
        {#snippet action()}
            <Button variant="secondary" onclick={onRetry}><span>Retry</span
                ></Button>
        {/snippet}
    </EmptyState>
{:else if !stage || status === "error"}
    <EmptyState
        tone="empty"
        title="Stage not found"
        description="The requested stage is not available in this project."
    >
        {#snippet icon()}<ui5-icon name="question-mark" aria-hidden="true"
            ></ui5-icon>{/snippet}
    </EmptyState>
{:else}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="flex min-w-0 flex-col gap-4">
            <StageVersionDetails
                label="Target"
                version={stage.targetStageVersion}
            />
            {#if projectId !== undefined && targetVersionId !== undefined}
                <VectorDeploymentSection
                    {projectId}
                    landscapeId={stage.landscapeId}
                    vectorDeploymentId={targetVersionId}
                />
            {/if}
        </div>
        <div class="flex min-w-0 flex-col gap-4">
            <StageVersionDetails
                label="Active"
                version={stage.activeStageVersion}
            />
            {#if !collapsed && projectId !== undefined && activeVersionId !== undefined}
                <VectorDeploymentSection
                    {projectId}
                    landscapeId={stage.landscapeId}
                    vectorDeploymentId={activeVersionId}
                />
            {/if}
        </div>
    </div>
{/if}
