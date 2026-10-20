<script lang="ts">
    import { page } from "$app/state";
    import { Breadcrumbs, Button, PageHeader } from "@konfidence/design-system/components";
    import type { BreadcrumbItem } from "@konfidence/design-system/components";
    import StageDetails from "#lib/landscape/components/StageDetails.svelte";
    import { projectLandscapeUrl } from "#lib/projects/url.ts";
    import { useStage } from "#lib/queries.svelte.ts";
    import { isEmbedded } from "#lib/shell/embedded.ts";
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
    const backHref = $derived(projectLandscapeUrl(data.projectId, isEmbedded(page.url)));
    const stageName = $derived(stage?.name ?? data.stageId);
    const crumbs: BreadcrumbItem[] = $derived([
        { href: backHref, label: data.projectId },
        { label: data.landscapeId },
        { label: stageName },
    ]);
</script>

<svelte:head><title>{stageName} · Konfidence</title></svelte:head>

<section class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10">
    <PageHeader
        title={stageName}
        description={`Project ${data.projectId} · Landscape ${data.landscapeId}`}
    >
        {#snippet eyebrow()}
            <Breadcrumbs items={crumbs} />
        {/snippet}
        {#snippet actions()}
            <Button href={backHref} variant="secondary">Back to landscape</Button>
        {/snippet}
    </PageHeader>
    <StageDetails
        error={error?.message}
        errorStatus={error?.status}
        onRetry={reload}
        projectId={data.projectId}
        {stage}
        {status}
    />
</section>
