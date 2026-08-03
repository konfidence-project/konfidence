<script lang="ts">
    import ErrorView from "$lib/components/ErrorView.svelte";
    import LandscapeView from "$lib/components/LandscapeView.svelte";
    import LoadingView from "$lib/components/LoadingView.svelte";
    import { getProjectLandscape } from "$lib/konfidence-api/queries.remote";
    import { toLandscapeView } from "$lib/stages";
    import type { PageProps } from "./$types";

    const { params }: PageProps = $props();
    const landscapeQuery = $derived(getProjectLandscape(params.projectId));
    const landscape = $derived.by(() => {
        if (!landscapeQuery.ready) {
            return undefined;
        }
        return toLandscapeView(
            landscapeQuery.current.landscapes,
            landscapeQuery.current.stages,
        );
    });
</script>

{#if landscapeQuery.loading && !landscape}
    <LoadingView title="Loading landscape" message="Loading landscapes and stages." />
{:else if landscapeQuery.error}
    <div class="grid justify-items-center gap-4 p-8">
        <ErrorView
            title="Failed to load landscape"
            message="The project landscape is currently unavailable."
            error={landscapeQuery.error}
        />
        <button class="btn preset-filled-primary-500" type="button" onclick={() => landscapeQuery.refresh()}>Retry</button>
    </div>
{:else if landscape}
    <LandscapeView landscapes={landscape.landscapes} stages={landscape.stages} />
{/if}
