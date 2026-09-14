<script lang="ts">
    import { page } from "$app/state";
    import { getApiClient } from "$lib/konfidence-api/client-instance";
    import LandscapeOverview from "$lib/landscape/components/LandscapeOverview.svelte";
    import { LandscapeDataStore } from "$lib/landscape/landscape-data.svelte";
    import { isEmbedded } from "$lib/shell/embedded";

    const projectId = $derived(page.params.projectId ?? "");
    const embedded = $derived(isEmbedded(page.url));
    const store = new LandscapeDataStore(getApiClient());

    $effect(() => {
        void store.load(projectId);
        return () => store.cancel();
    });
</script>

<svelte:head>
    <title>Landscape · Konfidence</title>
</svelte:head>

<div class="h-full min-h-0">
    <LandscapeOverview
        error={store.error}
        {embedded}
        landscapes={store.landscapes}
        onRetry={() => void store.load(projectId)}
        {projectId}
        stages={store.stages}
        status={store.status}
    />
</div>
