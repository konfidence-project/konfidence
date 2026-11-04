<script lang="ts">
    import { useLandscapes, useVectorPromotionConfigs } from "#lib/queries.svelte.ts";
    import PromotionsView from "./PromotionsView.svelte";
    import { usePromotionRefresh } from "./use-promotion-refresh.ts";

    interface Props { projectId: string; }
    let { projectId }: Props = $props();
    const configsQuery = useVectorPromotionConfigs(() => ({ projectId }));
    const landscapesQuery = useLandscapes(() => ({ projectId }));
    usePromotionRefresh(configsQuery.reload);
</script>

<PromotionsView configs={configsQuery.data ?? []} landscapes={landscapesQuery.data ?? []}
    loading={configsQuery.loading} hasLoaded={configsQuery.data !== undefined}
    error={configsQuery.error?.message} onRetry={configsQuery.reload} />
