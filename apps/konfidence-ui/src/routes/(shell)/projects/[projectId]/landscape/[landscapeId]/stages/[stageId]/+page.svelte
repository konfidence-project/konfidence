<script lang="ts">
    import { page } from "$app/state";
    import { getApiClient } from "$lib/konfidence-api/client-instance";
    import StageDetails from "$lib/landscape/components/StageDetails.svelte";
    import { LandscapeDataStore } from "$lib/landscape/landscape-data.svelte";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";

    const projectId = $derived(page.params.projectId ?? "");
    const landscapeId = $derived(page.params.landscapeId ?? "");
    const stageId = $derived(page.params.stageId ?? "");
    const embedded = $derived(isEmbedded(page.url));
    const store = new LandscapeDataStore(getApiClient());
    const landscape = $derived(
        store.landscapes.find(({ id }) => id === landscapeId),
    );
    const stage = $derived(
        store.stages.find(
            ({ id, landscapeId: owner }) =>
                id === stageId && owner === landscapeId,
        ),
    );

    // Fetch on route-ID changes; cancel the previous request on navigation or unmount.
    $effect(() => {
        void store.load(projectId, landscapeId);
        return () => store.cancel();
    });
</script>

<svelte:head><title>{stage?.name ?? "Stage"} · Konfidence</title></svelte:head>

<section class="mx-auto max-w-280 px-4 py-6 sm:px-6 sm:py-8">
    <StageDetails
        backHref={projectLandscapeUrl(projectId, embedded)}
        error={store.error}
        errorStatus={store.errorStatus}
        {landscape}
        onRetry={() => void store.load(projectId, landscapeId)}
        {stage}
        status={store.status}
    />
</section>
