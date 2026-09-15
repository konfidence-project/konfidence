<script lang="ts">
    import { page } from "$app/state";
    import LandscapeOverview from "$lib/landscape/components/LandscapeOverview.svelte";
    import { isEmbedded } from "$lib/shell/embedded";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    const embedded = $derived(isEmbedded(page.url));
</script>

<svelte:head>
    <title>Landscape · Konfidence</title>
</svelte:head>

<div class="h-full min-h-0">
    <LandscapeOverview
        error={data.store.error}
        {embedded}
        landscapes={data.store.landscapes}
        onRetry={() => void data.store.refresh(data.projectId)}
        projectId={data.projectId}
        stages={data.store.stages}
        status={data.store.status}
    />
</div>
