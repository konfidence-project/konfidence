<script lang="ts">
    import { page } from "$app/state";
    import StageDetails from "$lib/landscape/components/StageDetails.svelte";
    import { useLandscapes, useStages } from "$lib/queries.svelte.js";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    const landscapesQuery = useLandscapes(() => ({ projectId: data.projectId }));
    const stagesQuery = useStages(() => ({
        landscapeId: data.landscapeId,
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
    };
    const embedded = $derived(isEmbedded(page.url));
    const landscape = $derived(
        landscapesQuery.data?.find(({ id }) => id === data.landscapeId),
    );
    const stage = $derived(
        stagesQuery.data?.find(
            ({ id, landscapeId: owner }) =>
                id === data.stageId && owner === data.landscapeId,
        ),
    );
</script>

<svelte:head><title>{stage?.name ?? "Stage"} · Konfidence</title></svelte:head>

<section class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10">
    <StageDetails
        backHref={projectLandscapeUrl(data.projectId, embedded)}
        error={error?.message}
        errorStatus={error?.status}
        {landscape}
        onRetry={reload}
        {stage}
        {status}
    />
</section>
