<script lang="ts">
    import StageDetails from "#lib/landscape/components/StageDetails.svelte";
    import { useStage } from "#lib/queries.svelte.ts";
    import type { PageProps } from "./$types";

    let { data }: PageProps = $props();

    const stageQuery = useStage(() => ({
        landscapeId: data.landscapeId,
        projectId: data.projectId,
        stageId: data.stageId,
    }));
    const loading = $derived(stageQuery.loading);
    const error = $derived(stageQuery.error);
    const status = $derived.by(() => {
        if (loading) {
            return "loading";
        }
        return error ? "error" : "ready";
    });
    const reload = (): void => {
        stageQuery.reload();
    };
    const stage = $derived(stageQuery.data ?? undefined);
</script>

<svelte:head><title>{stage?.name ?? "Stage"} · Konfidence</title></svelte:head>

<section class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10">
    <StageDetails
        error={error?.message}
        errorStatus={error?.status}
        onRetry={reload}
        projectId={data.projectId}
        {stage}
        {status}
    />
</section>
