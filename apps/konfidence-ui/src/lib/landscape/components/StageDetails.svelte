<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents-icons/dist/question-mark.js";
    import {
        Breadcrumbs,
        Button,
        EmptyState,
        OrbitLoader,
        PageHeader,
    } from "@konfidence/design-system/components";
    import type { BreadcrumbItem } from "@konfidence/design-system/components";
    import type { Landscape, Stage } from "$lib/landscape/landscapeApi";
    import type { LandscapeDataStatus } from "$lib/landscape/landscapeData.svelte";
    import StageVersionDetails from "$lib/landscape/components/StageVersionDetails.svelte";

    interface Props {
        backHref: string;
        error?: string;
        errorStatus?: number;
        landscape?: Landscape;
        stage?: Stage;
        status: LandscapeDataStatus;
        onRetry: () => void;
    }
    let {
        backHref,
        error,
        errorStatus,
        landscape,
        stage,
        status,
        onRetry,
    }: Props = $props();

    const crumbs: BreadcrumbItem[] = $derived(
        landscape && stage
            ? [
                  { href: backHref, label: "Landscapes" },
                  // No dedicated per-landscape route yet — this crumb
                  // renders as plain text and becomes a link when the
                  // landscape detail route ships.
                  { label: landscape.name },
                  { label: stage.name },
              ]
            : [],
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
{:else if !landscape || !stage || status === "error"}
    <EmptyState
        tone="empty"
        title="Stage not found"
        description="The requested landscape or stage is not available in this project."
    >
        {#snippet icon()}<ui5-icon name="question-mark" aria-hidden="true"
            ></ui5-icon>{/snippet}
        {#snippet action()}
            <Button href={backHref}><span>Back to landscapes</span></Button>
        {/snippet}
    </EmptyState>
{:else}
    <PageHeader title={stage.name}>
        {#snippet eyebrow()}
            <Breadcrumbs items={crumbs} />
        {/snippet}
    </PageHeader>
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <StageVersionDetails
            label="Target"
            version={stage.targetStageVersion}
        /><StageVersionDetails
            label="Active"
            version={stage.activeStageVersion}
        />
    </div>
{/if}
