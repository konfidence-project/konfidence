<script lang="ts">
    import "@ui5/webcomponents-icons/dist/product.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import type {components} from "@konfidence/api-client/schema";
    import {
        Button,
        EmptyState,
        Link,
        OrbitLoader,
        PageHeader,
        SearchInput,
        Select,
        SidePanel,
    } from "@konfidence/design-system/components";

    import VectorDeploymentDetail from "./VectorDeploymentDetail.svelte";
    import VectorDeploymentsTable from "./VectorDeploymentsTable.svelte";
    import {toVectorDeploymentRows} from "./deployments.js";
    import {useDeploymentFilters} from "./useDeploymentFilters.svelte.js";

    type Landscape = components["schemas"]["Landscape"];
    type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
    type Stage = components["schemas"]["Stage"];
    type VectorDeployment = components["schemas"]["VectorDeployment"];

    interface Props {
        projectId: string;
        artifactDeployments?: readonly ArtifactDeployment[];
        landscapes?: readonly Landscape[];
        stages?: readonly Stage[];
        vectorDeployments?: readonly VectorDeployment[];
        selectedLandscapeId?: string | undefined;
        loading?: boolean;
        hasLoaded?: boolean;
        error?: string | undefined;
        onRetry?: () => void;
    }

    let {
        projectId,
        artifactDeployments = [],
        landscapes = [],
        stages = [],
        vectorDeployments = [],
        selectedLandscapeId,
        loading = false,
        hasLoaded = false,
        error,
        onRetry,
    }: Props = $props();

    const rows = $derived(
        toVectorDeploymentRows({ artifactDeployments, landscapes, stages, vectorDeployments }),
    );

    const filters = useDeploymentFilters({
        rows: () => rows,
        selectedLandscapeId: () => selectedLandscapeId,
    });

    const labelTextClass =
        "text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]";
</script>

<section
        class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10"
        aria-labelledby="vectordeployment-view-title"
>
    <PageHeader
            id="vectordeployment-view-title"
            title="Vector Deployments"
    />

    {#if hasLoaded && !error}
        <div
                class="flex flex-wrap items-end gap-3 rounded-[var(--card-radius)] border border-[color:var(--border-subtle)] bg-[color:var(--surface-subtle)] px-4 py-3"
                data-testid="vectordeployment-view-filters"
        >
            <div class="flex min-w-[16rem] grow-[2] basis-[20rem] flex-col gap-1">
                <span class={labelTextClass}>Search</span>
                <SearchInput
                        bind:value={filters.query}
                        placeholder="Search artifact, version, stage, vector…"
                        aria-label="Search vector deployments"
                        data-testid="vectordeployment-search"
                        class="w-full max-w-none"
                />
            </div>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Status</span>
                <Select
                        bind:value={filters.status}
                        aria-label="Filter by status"
                        data-testid="vectordeployment-status-filter"
                >
                    <option value="">All statuses</option>
                    <option value="DeployingVector">Deploying</option>
                    <option value="DeploymentReady">Ready</option>
                    <option value="DeploymentFailed">Failed</option>
                </Select>
            </label>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Landscape</span>
                <Select
                        value={selectedLandscapeId ?? ""}
                        onchange={(event) => filters.changeLandscape(event.currentTarget.value)}
                        disabled={landscapes.length === 0}
                        aria-label="Filter by landscape"
                        data-testid="vectordeployment-landscape-filter"
                >
                    <option value="">All landscapes</option>
                    {#each landscapes as landscape (landscape.id)}
                        <option value={landscape.id}>{landscape.name}</option>
                    {/each}
                </Select>
            </label>
            <div class="flex min-w-0 flex-col">
                <span class={labelTextClass} aria-hidden="true">&nbsp;</span>
                <div class="flex items-center border border-transparent px-1.5 py-2 leading-[1.4]">
                    <Link
                            class={!filters.anyFiltersActive ? "invisible pointer-events-none" : undefined}
                            onclick={filters.clearFilters}
                            disabled={!filters.anyFiltersActive}
                            aria-hidden={!filters.anyFiltersActive}
                            tabindex={filters.anyFiltersActive ? 0 : -1}
                            data-testid="vectordeployment-view-clear-filters"
                    >
                        Clear filters
                    </Link>
                </div>
            </div>
        </div>
    {/if}

    {#if loading}
        <div class="flex justify-center py-12" data-testid="vectordeployment-view-loading">
            <OrbitLoader label="Loading vector deployments…"/>
        </div>
    {:else if error}
        <div data-testid="vectordeployment-view-error">
            <EmptyState tone="error" title="Failed to load vector deployments" description={error}>
                {#snippet icon()}
                    <ui5-icon name="error" aria-hidden="true"></ui5-icon>
                {/snippet}
                {#snippet action()}
                    {#if onRetry}
                        <Button variant="secondary" onclick={onRetry} data-testid="vectordeployment-view-retry">
                            <span>Retry</span>
                        </Button>
                    {/if}
                {/snippet}
            </EmptyState>
        </div>
    {:else if rows.length === 0}
        <div data-testid="vectordeployment-view-empty">
            <EmptyState
                    title={filters.serverFiltersActive
                    ? "No vector deployments match the current filters"
                    : "No vector deployments"}
                    description={filters.serverFiltersActive
                    ? "Try widening the landscape filter."
                    : "This project does not have any vector deployments yet."}
            >
                {#snippet icon()}
                    <ui5-icon name="product" aria-hidden="true"></ui5-icon>
                {/snippet}
                {#snippet action()}
                    {#if filters.serverFiltersActive}
                        <Link onclick={filters.clearFilters} data-testid="vectordeployment-view-empty-clear">
                            Clear filters
                        </Link>
                    {/if}
                {/snippet}
            </EmptyState>
        </div>
    {:else}
        <div class="flex flex-col gap-2">
            <p
                    class="m-0 text-[length:var(--text-sm)] text-[color:var(--text-tertiary)]"
                    aria-live="polite"
                    data-testid="vectordeployment-view-count"
            >
                {filters.visibleRows.length.toLocaleString()} of {rows.length.toLocaleString()}
                deployment{rows.length === 1 ? "" : "s"}
            </p>
            {#if filters.visibleRows.length === 0}
                <div
                        class="flex flex-col items-center gap-2 rounded-[var(--card-radius)] border border-dashed border-[color:var(--border-default)] px-6 py-12 text-center text-[color:var(--text-secondary)]"
                        role="status"
                        data-testid="vectordeployment-table-no-results"
                >
                    <p class="m-0">No vector deployments match the current filters.</p>
                    <Link onclick={filters.clearFilters}>
                        Clear filters
                    </Link>
                </div>
            {:else}
                <VectorDeploymentsTable {projectId} rows={filters.visibleRows} selectedId={filters.selectedId}
                                        onSelect={filters.openRow}/>
            {/if}
        </div>
    {/if}
</section>

<SidePanel
        open={filters.panelOpen}
        title={filters.selected ? `${filters.selected.stageName}/${filters.selected.id}` : "Vector deployment"}
        onClose={filters.closePanel}
>
    {#if filters.selected}
        <VectorDeploymentDetail row={filters.selected}/>
    {/if}
</SidePanel>
