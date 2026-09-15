<script lang="ts">
    import {
        Button,
        OrbitLoader,
        PageHeader,
    } from "@konfidence/design-system/components";
    import type { Landscape, Stage } from "$lib/landscape/landscapeApi";
    import type { LandscapeDataStatus } from "$lib/landscape/landscapeData.svelte";
    import LandscapeFlow from "$lib/landscape/components/LandscapeFlow.svelte";

    interface Props {
        error?: string;
        status: LandscapeDataStatus;
        landscapes: readonly Landscape[];
        stages: readonly Stage[];
        projectId: string;
        embedded: boolean;
        onRetry: () => void;
    }
    let {
        error,
        status,
        landscapes,
        stages,
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
        <PageHeader id="landscape-title" title="Landscapes" />
    </div>
    {#if status === "loading"}
        <div class="place-self-center p-6">
            <OrbitLoader label="Loading landscapes and stages" />
        </div>
    {:else if status === "error"}
        <div
            class="place-self-center p-6 text-center text-[var(--text-secondary)]"
            role="alert"
        >
            <h2>Landscape could not be loaded</h2>
            <p>{error}</p>
            <Button onclick={onRetry}>Retry</Button>
        </div>
    {:else if landscapes.length === 0 && stages.length === 0}
        <div
            class="place-self-center p-6 text-center text-[var(--text-secondary)]"
        >
            <h2>No landscapes yet</h2>
            <p>This project has no landscapes or stages to display.</p>
        </div>
    {:else}
        <LandscapeFlow {landscapes} {stages} {projectId} {embedded} />
    {/if}
</section>
