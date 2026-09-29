<script lang="ts">
    import {
        Button,
        OrbitLoader,
        PageHeader,
    } from "@konfidence/design-system/components";
    import type { Landscape, Stage, VectorPromotionConfig } from "$lib/konfidence-api/types";
    import LandscapeFlow from "$lib/landscape/components/LandscapeFlow.svelte";

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
</script>

<section
    class="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]"
    aria-labelledby="landscape-title"
>
    <!-- Header is inset by the standard page gutter; the flow canvas below
         keeps its edge-to-edge working area for panning. -->
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
        />
    {/if}
</section>
