<script lang="ts">
    import {
        Breadcrumbs,
        Button,
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
    <section role="alert">
        <h1>Stage details could not be loaded</h1>
        <p>{error}</p>
        <Button onclick={onRetry}>Retry</Button>
    </section>
{:else if !landscape || !stage || status === "error"}
    <section>
        <h1>Stage not found</h1>
        <p>
            The requested landscape or stage is not available in this project.
        </p>
        <Button href={backHref}>Back to landscapes</Button>
    </section>
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
