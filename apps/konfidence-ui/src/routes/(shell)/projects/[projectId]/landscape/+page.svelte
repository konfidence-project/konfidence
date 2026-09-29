<script lang="ts">
    import { page } from "$app/state";
    import LandscapeOverview from "$lib/landscape/components/LandscapeOverview.svelte";
    import {
        useLandscapes,
        useStages,
        useVectorPromotionConfigs,
    } from "$lib/queries.svelte.js";
    import { isEmbedded } from "$lib/shell/embedded";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    const landscapesQuery = useLandscapes(() => ({ projectId: data.projectId }));
    const stagesQuery = useStages(() => ({ projectId: data.projectId }));
    const promotionsQuery = useVectorPromotionConfigs(() => ({
        projectId: data.projectId,
    }));
    const loading = $derived(landscapesQuery.loading || stagesQuery.loading);
    const error = $derived(landscapesQuery.error ?? stagesQuery.error);
    const status = $derived.by(() => {
        if (loading) {
            return "loading";
        }
        return error ? "error" : "ready";
    });
    const reload = (): void => {
        landscapesQuery.reload();
        stagesQuery.reload();
        promotionsQuery.reload();
    };
    const embedded = $derived(isEmbedded(page.url));
</script>

<svelte:head>
    <title>Landscape · Konfidence</title>
</svelte:head>

<div class="h-full min-h-0">
    <LandscapeOverview
        error={error?.message}
        {embedded}
        landscapes={landscapesQuery.data ?? []}
        onRetry={reload}
        projectId={data.projectId}
        promotionConfigs={promotionsQuery.data ?? []}
        stages={stagesQuery.data ?? []}
        {status}
    />
</div>
