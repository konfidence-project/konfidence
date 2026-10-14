<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import {
        Button,
        OrbitLoader,
        PageHeader,
        SidePanel,
    } from "@konfidence/design-system/components";
    import type { Landscape, Stage, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
    import LandscapeFlow from "#lib/landscape/components/LandscapeFlow.svelte";
    import StageDetails from "#lib/landscape/components/StageDetails.svelte";
    import { STAGE_PARAM } from "#lib/landscape/params.ts";
    import { stageDetailsUrl } from "#lib/projects/url.ts";

    interface Props {
        error?: string;
        status: "loading" | "ready" | "error";
        landscapes: readonly Landscape[];
        stages: readonly Stage[];
        promotionConfigs: readonly VectorPromotionConfig[];
        projectId: string;
        embedded: boolean;
        onRetry: () => void;
    }
    let {
        error,
        status,
        landscapes,
        stages,
        promotionConfigs,
        projectId,
        embedded,
        onRetry,
    }: Props = $props();

    const selectedStageId = $derived(page.url.searchParams.get(STAGE_PARAM) ?? undefined);
    const selectedStage = $derived(
        selectedStageId === undefined
            ? undefined
            : stages.find((candidate) => candidate.id === selectedStageId),
    );

    const closeOverlay = (): void => {
        if (!page.url.searchParams.has(STAGE_PARAM)) {
            return;
        }
        const next = new URL(page.url);
        next.searchParams.delete(STAGE_PARAM);
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- reusing the current page URL.
        void goto(next, { keepFocus: true, noScroll: true });
    };

    const openStageDetails = (): void => {
        if (selectedStage === undefined) {
            return;
        }
        const href = stageDetailsUrl({
            embedded,
            landscapeId: selectedStage.landscapeId,
            projectId,
            stageId: selectedStage.id,
        });
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- the shared helper resolves this typed application route.
        void goto(href);
    };
</script>

<section
    class="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]"
    aria-labelledby="landscape-title"
>
    <div class="px-6 pt-6">
        <PageHeader
            id="landscape-title"
            title="Landscape"
        />
    </div>
    {#if status === "loading"}
        <div class="place-self-center p-6">
            <OrbitLoader label="Loading landscapes and stages" />
        </div>
    {:else if status === "error"}
        <div
            class="place-self-center p-6 text-center text-content-secondary"
            role="alert"
        >
            <h2>Landscape could not be loaded</h2>
            <p>{error}</p>
            <Button onclick={onRetry}>Retry</Button>
        </div>
    {:else if landscapes.length === 0}
        <div
            class="place-self-center p-6 text-center text-content-secondary"
        >
            <h2>No landscapes yet</h2>
            <p>This project has no landscapes or stages to display.</p>
        </div>
    {:else}
        <LandscapeFlow
            {landscapes}
            {stages}
            {promotionConfigs}
            {projectId}
            {embedded}
            {selectedStageId}
        />
    {/if}
</section>

<SidePanel
    open={selectedStage !== undefined}
    title={selectedStage?.name ?? "Stage"}
    onClose={closeOverlay}
    onResize={openStageDetails}
>
    {#if selectedStage}
        <StageDetails {projectId} stage={selectedStage} status="ready" onRetry={() => undefined} />
    {/if}
</SidePanel>
