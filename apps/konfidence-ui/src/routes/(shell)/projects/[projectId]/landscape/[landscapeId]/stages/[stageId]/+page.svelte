<script lang="ts">
    import { page } from "$app/state";
    import StageDetails from "$lib/landscape/components/StageDetails.svelte";
    import { projectLandscapeUrl } from "$lib/projects/url";
    import { isEmbedded } from "$lib/shell/embedded";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    const embedded = $derived(isEmbedded(page.url));
    const landscape = $derived(
        data.store.landscapes.find(({ id }) => id === data.landscapeId),
    );
    const stage = $derived(
        data.store.stages.find(
            ({ id, landscapeId: owner }) =>
                id === data.stageId && owner === data.landscapeId,
        ),
    );
</script>

<svelte:head><title>{stage?.name ?? "Stage"} · Konfidence</title></svelte:head>

<section class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10">
    <StageDetails
        backHref={projectLandscapeUrl(data.projectId, embedded)}
        error={data.store.error}
        errorStatus={data.store.errorStatus}
        {landscape}
        onRetry={() =>
            void data.store.refresh(data.projectId, {
                landscapeId: data.landscapeId,
            })}
        {stage}
        status={data.store.status}
    />
</section>
