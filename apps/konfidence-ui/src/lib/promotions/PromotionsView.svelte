<script lang="ts">
    import { Button, EmptyState, ExpandableGroup, Link, OrbitLoader, PageHeader, SearchInput, Select } from "@konfidence/design-system/components";
    import type { Landscape, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
    import { isActionable, referenceLabel } from "./promotions.ts";
    import { compatibleOptions, filterPromotionGroups } from "./filters.ts";
    import { usePromotionFilters } from "./usePromotionFilters.svelte.ts";
    import PromotionRow from "./PromotionRow.svelte";
    import { vectorLabel } from "./vector-label.ts";

    interface Props {
        configs?: readonly VectorPromotionConfig[];
        landscapes?: readonly Landscape[];
        loading?: boolean;
        hasLoaded?: boolean;
        error?: string;
        onRetry: () => void;
    }
    let { configs = [], landscapes = [], loading = false, hasLoaded = false, error, onRetry }: Props = $props();
    let expanded = $state<Record<string, boolean>>({});
    let selectedId = $state<string>();
    const filters = usePromotionFilters();
    const groups = $derived(filterPromotionGroups(configs, landscapes, filters.values));
    const options = $derived(compatibleOptions(configs, landscapes, filters.values));
    const labelClass = "text-meta font-semibold uppercase tracking-[0.03em] text-content-tertiary";
</script>

<section class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10" aria-labelledby="promotions-view-title">
    <PageHeader id="promotions-view-title" title="Promotions" />
    {#if hasLoaded}
        {#if error}
            <div data-promotion-refresh role="alert" class="flex flex-wrap items-center gap-3 rounded-[var(--card-radius)] border border-outline-subtle bg-status-error-bg p-3 text-compact text-status-error-fg">
                <span>Could not refresh promotions. Showing previously loaded data. {error}</span>
                <Button variant="secondary" onclick={onRetry}>Retry</Button>
            </div>
        {/if}
        <div class="flex flex-wrap items-end gap-3 rounded-[var(--card-radius)] border border-outline-subtle bg-surface-subtle px-4 py-3">
            <label class="flex min-w-0 grow basis-64 flex-col gap-1">
                <span class={labelClass}>Search</span>
                <SearchInput bind:value={filters.query} aria-label="Search promotions" placeholder="Search configuration, promotion, vector, approval…" class="w-full max-w-none" />
            </label>
            <label class="flex min-w-0 grow basis-48 flex-col gap-1">
                <span class={labelClass}>Landscape</span>
                <Select aria-label="Filter by landscape" value={filters.values.landscape} onchange={(event) => filters.change("promotionLandscape", event.currentTarget.value)}>
                    <option value="">All landscapes</option>
                    {#each options.landscapes as option (option.value)}<option value={option.value}>{option.label}</option>{/each}
                </Select>
            </label>
            <label class="flex min-w-0 grow basis-48 flex-col gap-1">
                <span class={labelClass}>Stage</span>
                <Select aria-label="Filter by stage" value={filters.values.stage} onchange={(event) => filters.change("promotionStage", event.currentTarget.value)}>
                    <option value="">All stages</option>
                    {#each options.stages as option (option.value)}<option value={option.value}>{option.label}</option>{/each}
                </Select>
            </label>
            <label class="flex min-w-0 grow basis-48 flex-col gap-1">
                <span class={labelClass}>PromotionConfig</span>
                <Select aria-label="Filter by promotion configuration" value={filters.values.config} onchange={(event) => filters.change("promotionConfig", event.currentTarget.value)}>
                    <option value="">All configurations</option>
                    {#each options.configs as option (option.value)}<option value={option.value}>{option.label}</option>{/each}
                </Select>
            </label>
            <div class="flex min-w-0 flex-col">
                <span class={labelClass} aria-hidden="true">&nbsp;</span>
                <div class="flex items-center border border-transparent px-1.5 py-2 text-compact leading-[1.4]">
                    <Link onclick={filters.clear} disabled={!filters.active}
                        class={!filters.active ? "invisible pointer-events-none" : undefined}
                        aria-hidden={!filters.active} tabindex={filters.active ? 0 : -1}
                        data-testid="promotion-view-clear-filters">Clear filters</Link>
                </div>
            </div>
        </div>
    {/if}
    {#if !hasLoaded && loading}
        <div class="flex justify-center py-12"><OrbitLoader label="Loading promotions…" /></div>
    {:else if error && !hasLoaded}
        <EmptyState tone="error" title="Failed to load promotions" description={error}>
            {#snippet action()}<Button variant="secondary" onclick={onRetry}>Retry</Button>{/snippet}
        </EmptyState>
    {:else if configs.length === 0}
        <EmptyState title="No promotion configurations" description="This project does not have any configured promotion flows yet." />
    {:else}
        {#if groups.some((group) => group.promotions.some(isActionable))}
            <p class="m-0 text-meta text-content-secondary">Approval and rejection are unavailable in the dashboard in this release.</p>
        {/if}
        <p class="m-0 text-compact text-content-tertiary" aria-live="polite">{groups.length} of {configs.length} configurations</p>
        {#if groups.length === 0}
            <EmptyState title="No promotions match the current filters" description="Try widening your filters or search.">
                {#snippet action()}<Link onclick={filters.clear}>Clear filters</Link>{/snippet}
            </EmptyState>
        {/if}
        <div class="flex flex-col gap-4">
            {#each groups as group (group.config.id)}
                {#snippet history()}
                    {#each group.promotions as promotion (promotion.id)}
                        <PromotionRow {promotion} />
                    {/each}
                {/snippet}
                <ExpandableGroup ariaLabel={`Promotion configuration ${group.config.id}`} triggerId={`promotion-group-${group.config.id}`}
                    bind:expanded={() => expanded[group.config.id] ?? false, (value) => expanded[group.config.id] = value}
                    bind:selectedValue={selectedId}
                    children={group.promotions.length ? history : undefined}>
                    {#snippet header()}
                        <span class="flex min-w-0 flex-col gap-2">
                            <span class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                                <span class="text-base font-semibold">{referenceLabel(group.config.source, landscapes)} → {referenceLabel(group.config.target, landscapes)}</span>
                                <span class="text-meta text-content-tertiary">{group.config.id}</span>
                            </span>
                            <span class="flex flex-wrap items-center gap-x-3 gap-y-2 text-compact text-content-secondary">
                                <span>{group.waiting} waiting</span>
                                {#if group.latestWaiting}
                                    <button type="button" aria-label={`Latest waiting promotion ${group.latestWaiting.id}`}
                                        onclick={(event) => { event.stopPropagation(); expanded[group.config.id] = true; selectedId = group.latestWaiting?.id; }}
                                        class="inline-flex min-w-0 max-w-full cursor-pointer items-center gap-1 rounded-[var(--radius-sm)] border border-[var(--tag-teal-bd)] bg-[var(--tag-teal-bg)] px-2 py-1 text-[var(--tag-teal-fg)] focus-visible:outline-2 focus-visible:outline-outline-focus sm:max-w-96">
                                        <span class="shrink-0">Latest waiting:</span>
                                        <span title={group.latestWaiting.vector || group.latestWaiting.id} class="min-w-0 truncate font-mono">{vectorLabel(group.latestWaiting.vector) || group.latestWaiting.id}</span>
                                    </button>
                                {:else if group.promotions.length === 0}
                                    <span>{group.config.promotions.length ? "No matching promotions" : "No promotions yet"}</span>
                                {/if}
                                <span class="text-meta">{group.promotions.length}{group.promotions.length !== group.config.promotions.length ? ` of ${group.config.promotions.length} matching` : ""} promotion{group.config.promotions.length === 1 ? "" : "s"}</span>
                            </span>
                        </span>
                    {/snippet}
                </ExpandableGroup>
            {/each}
        </div>
    {/if}
</section>
