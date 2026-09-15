<script lang="ts">
    import { Button, OrbitLoader } from "@konfidence/design-system/components";
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
    <h1
        class="m-0 border-b border-[var(--border-subtle)] px-6 py-5 text-[length:var(--text-h2)] font-semibold text-[var(--text-primary)]"
        id="landscape-title"
        data-testid="page-heading"
    >
        Landscapes
    </h1>
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
