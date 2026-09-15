<script lang="ts">
    import "@ui5/webcomponents-icons/dist/product.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import type { components } from "@konfidence/api-client/schema";
    import {
        Button,
        EmptyState,
        OrbitLoader,
        PageHeader,
        SearchInput,
        Select,
        SidePanel,
    } from "@konfidence/design-system/components";

    import ArtifactDeploymentDetail from "./ArtifactDeploymentDetail.svelte";
    import ArtifactDeploymentsTable from "./ArtifactDeploymentsTable.svelte";
    import { toArtifactDeploymentRows } from "./deployments.js";
    import { useDeploymentFilters } from "./useDeploymentFilters.svelte.js";

    type Landscape = components["schemas"]["Landscape"];
    type ArtifactDeployment = components["schemas"]["ArtifactDeployment"];
    type Stage = components["schemas"]["Stage"];
    type VectorDeployment = components["schemas"]["VectorDeployment"];

    interface Props {
        artifactDeployments?: readonly ArtifactDeployment[];
        landscapes?: readonly Landscape[];
        stages?: readonly Stage[];
        vectorDeployments?: readonly VectorDeployment[];
        selectedLandscapeId?: string | undefined;
        selectedVectorDeploymentId?: string | undefined;
        loading?: boolean;
        hasLoaded?: boolean;
        error?: string | undefined;
        onRetry?: () => void;
    }

    let {
        artifactDeployments = [],
        landscapes = [],
        stages = [],
        vectorDeployments = [],
        selectedLandscapeId,
        selectedVectorDeploymentId,
        loading = false,
        hasLoaded = false,
        error,
        onRetry,
    }: Props = $props();

    const rows = $derived(
        toArtifactDeploymentRows({ artifactDeployments, landscapes, stages, vectorDeployments }),
    );

    const filters = useDeploymentFilters({
        rows: () => rows,
        selectedLandscapeId: () => selectedLandscapeId,
        selectedVectorDeploymentId: () => selectedVectorDeploymentId,
    });

    const vectorOptions = $derived(
        (selectedLandscapeId
            ? vectorDeployments.filter((deployment) => deployment.landscapeId === selectedLandscapeId)
            : vectorDeployments
        ).map((deployment) => ({
            id: deployment.id,
            label: `${deployment.vector.componentName}@${deployment.vector.componentVersion}`,
        })),
    );

    const labelTextClass =
        "text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]";
</script>

<section
    class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10"
    aria-labelledby="artifact-view-title"
>
    <PageHeader
        id="artifact-view-title"
        title="Artifact Deployments"
    />

    {#if hasLoaded && !error}
        <div
            class="flex flex-wrap items-end gap-3 rounded-[var(--card-radius)] border border-[color:var(--border-subtle)] bg-[color:var(--surface-subtle)] px-4 py-3"
            data-testid="artifact-view-filters"
        >
            <div class="flex min-w-[16rem] grow-[2] basis-[20rem] flex-col gap-1">
                <span class={labelTextClass}>Search</span>
                <SearchInput
                    bind:value={filters.query}
                    placeholder="Search artifact, version, stage, vector…"
                    aria-label="Search artifact deployments"
                    data-testid="artifact-search"
                    class="w-full max-w-none"
                />
            </div>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Status</span>
                <Select
                    bind:value={filters.status}
                    aria-label="Filter by status"
                    data-testid="artifact-status-filter"
                >
                    <option value="">All statuses</option>
                    <option value="ArtifactFetched">Fetched</option>
                    <option value="ArtifactDeployed">Deployed</option>
                    <option value="AppHealthy">Healthy</option>
                    <option value="Ready">Ready</option>
                </Select>
            </label>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Landscape</span>
                <Select
                    value={selectedLandscapeId ?? ""}
                    onchange={(event) => filters.changeLandscape(event.currentTarget.value)}
                    disabled={landscapes.length === 0}
                    aria-label="Filter by landscape"
                    data-testid="artifact-landscape-filter"
                >
                    <option value="">All landscapes</option>
                    {#each landscapes as landscape (landscape.id)}
                        <option value={landscape.id}>{landscape.name}</option>
                    {/each}
                </Select>
            </label>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Vector deployment</span>
                <Select
                    value={selectedVectorDeploymentId ?? ""}
                    onchange={(event) => filters.changeVectorDeployment(event.currentTarget.value)}
                    disabled={vectorOptions.length === 0}
                    aria-label="Filter by vector deployment"
                    data-testid="artifact-vector-filter"
                >
                    <option value="">All vector deployments</option>
                    {#each vectorOptions as option (option.id)}
                        <option value={option.id}>{option.label} · {option.id}</option>
                    {/each}
                </Select>
            </label>
            <div class="flex min-w-0 flex-col">
                <span class={labelTextClass} aria-hidden="true">&nbsp;</span>
                <button
                    type="button"
                    class={[
                        "cursor-pointer border border-transparent bg-transparent px-1.5 py-2 text-[length:var(--text-sm)] leading-[1.4] text-[color:var(--text-link,var(--btn-primary-fg))] underline underline-offset-[3px] focus-visible:rounded-[var(--radius-sm)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none",
                        !filters.anyFiltersActive && "invisible pointer-events-none",
                    ]}
                    onclick={filters.clearFilters}
                    disabled={!filters.anyFiltersActive}
                    aria-hidden={!filters.anyFiltersActive}
                    tabindex={filters.anyFiltersActive ? 0 : -1}
                    data-testid="artifact-view-clear-filters"
                >
                    Clear filters
                </button>
            </div>
        </div>
    {/if}

    {#if loading}
        <div class="flex justify-center py-12" data-testid="artifact-view-loading">
            <OrbitLoader label="Loading artifact deployments…" />
        </div>
    {:else if error}
        <div data-testid="artifact-view-error">
            <EmptyState tone="error" title="Failed to load artifact deployments" description={error}>
                {#snippet icon()}
                    <ui5-icon name="error" aria-hidden="true"></ui5-icon>
                {/snippet}
                {#snippet action()}
                    {#if onRetry}
                        <Button variant="secondary" onclick={onRetry} data-testid="artifact-view-retry">
                            <span>Retry</span>
                        </Button>
                    {/if}
                {/snippet}
            </EmptyState>
        </div>
    {:else if rows.length === 0}
        <div data-testid="artifact-view-empty">
            <EmptyState
                title={filters.serverFiltersActive
                    ? "No artifact deployments match the current filters"
                    : "No artifact deployments"}
                description={filters.serverFiltersActive
                    ? "Try widening the landscape or vector-deployment filter."
                    : "This project does not have any artifact deployments yet."}
            >
                {#snippet icon()}
                    <ui5-icon name="product" aria-hidden="true"></ui5-icon>
                {/snippet}
                {#snippet action()}
                    {#if filters.serverFiltersActive}
                        <Button variant="secondary" onclick={filters.clearFilters} data-testid="artifact-view-empty-clear">
                            <span>Clear filters</span>
                        </Button>
                    {/if}
                {/snippet}
            </EmptyState>
        </div>
    {:else}
        <div class="flex flex-col gap-2">
            <p
                class="m-0 text-[length:var(--text-sm)] text-[color:var(--text-tertiary)]"
                aria-live="polite"
                data-testid="artifact-view-count"
            >
                {filters.visibleRows.length.toLocaleString()} of {rows.length.toLocaleString()} deployment{rows.length === 1 ? "" : "s"}
            </p>
            {#if filters.visibleRows.length === 0}
                <div
                    class="flex flex-col items-center gap-2 rounded-[var(--card-radius)] border border-dashed border-[color:var(--border-default)] px-6 py-12 text-center text-[color:var(--text-secondary)]"
                    role="status"
                    data-testid="artifact-table-no-results"
                >
                    <p class="m-0">No artifact deployments match the current filters.</p>
                    <button
                        type="button"
                        class="cursor-pointer border-0 bg-transparent text-[color:var(--text-link,var(--btn-primary-fg))] underline underline-offset-[3px] focus-visible:rounded-[var(--radius-sm)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none"
                        onclick={filters.clearFilters}
                    >
                        Clear filters
                    </button>
                </div>
            {:else}
                <ArtifactDeploymentsTable rows={filters.visibleRows} selectedId={filters.selectedId} onSelect={filters.openRow} />
            {/if}
        </div>
    {/if}
</section>

<SidePanel
    open={filters.panelOpen}
    title={filters.selected ? `${filters.selected.component}@${filters.selected.version}` : "Artifact deployment"}
    onClose={filters.closePanel}
>
    {#if filters.selected}
        <ArtifactDeploymentDetail row={filters.selected} />
    {/if}
</SidePanel>
