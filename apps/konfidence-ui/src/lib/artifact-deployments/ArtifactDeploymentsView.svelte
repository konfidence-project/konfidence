<script lang="ts">
    import "@ui5/webcomponents-icons/dist/product.js";
    import "@ui5/webcomponents-icons/dist/error.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import type { components } from "@konfidence/api-client/schema";
    import {
        Button,
        EmptyState,
        OrbitLoader,
        SearchInput,
        SidePanel,
    } from "@konfidence/design-system/components";

    import ArtifactDeploymentDetail from "./ArtifactDeploymentDetail.svelte";
    import ArtifactDeploymentsTable from "./ArtifactDeploymentsTable.svelte";
    import type { ArtifactDeploymentRow, ArtifactDeploymentStatus } from "./deployments.js";
    import { matchesQuery, matchesStatus } from "./deployments.js";
    import { DEEP_LINK_PARAM, LANDSCAPE_PARAM, VECTOR_DEPLOYMENT_PARAM } from "./params.js";

    type Landscape = components["schemas"]["Landscape"];
    type VectorDeployment = components["schemas"]["VectorDeployment"];

    interface Props {
        rows: readonly ArtifactDeploymentRow[];
        landscapes?: readonly Landscape[];
        vectorDeployments?: readonly VectorDeployment[];
        selectedLandscapeId?: string | undefined;
        selectedVectorDeploymentId?: string | undefined;
        loading?: boolean;
        hasLoaded?: boolean;
        error?: string | undefined;
        onRetry?: () => void;
    }

    let {
        rows,
        landscapes = [],
        vectorDeployments = [],
        selectedLandscapeId,
        selectedVectorDeploymentId,
        loading = false,
        hasLoaded = false,
        error,
        onRetry,
    }: Props = $props();

    let query = $state("");
    let status = $state<"" | ArtifactDeploymentStatus>("");

    const visibleRows = $derived(
        rows.filter((row) => matchesQuery(row, query) && matchesStatus(row, status)),
    );

    const selectedId = $derived(page.url.searchParams.get(DEEP_LINK_PARAM) ?? undefined);
    const selected = $derived<ArtifactDeploymentRow | undefined>(
        selectedId ? rows.find((row) => row.id === selectedId) : undefined,
    );
    const panelOpen = $derived(selected !== undefined);

    const vectorOptions = $derived(
        (selectedLandscapeId
            ? vectorDeployments.filter((deployment) => deployment.landscapeId === selectedLandscapeId)
            : vectorDeployments
        ).map((deployment) => ({
            id: deployment.id,
            label: `${deployment.vector.componentName}@${deployment.vector.componentVersion}`,
        })),
    );

    const serverFiltersActive = $derived(
        selectedLandscapeId !== undefined || selectedVectorDeploymentId !== undefined,
    );
    const clientFiltersActive = $derived(query.trim() !== "" || status !== "");
    const anyFiltersActive = $derived(serverFiltersActive || clientFiltersActive);

    const updateUrl = (mutate: (params: URLSearchParams) => void): void => {
        const next = new globalThis.URL(page.url);
        mutate(next.searchParams);
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- reusing the current page URL.
        void goto(next, { keepFocus: true, noScroll: true });
    };

    const openRow = (row: ArtifactDeploymentRow): void => {
        if (page.url.searchParams.get(DEEP_LINK_PARAM) === row.id) {
            return;
        }
        updateUrl((params) => params.set(DEEP_LINK_PARAM, row.id));
    };

    const closePanel = (): void => {
        if (!page.url.searchParams.has(DEEP_LINK_PARAM)) {
            return;
        }
        updateUrl((params) => params.delete(DEEP_LINK_PARAM));
    };

    const changeLandscape = (value: string): void => {
        updateUrl((params) => {
            if (value === "") {
                params.delete(LANDSCAPE_PARAM);
            } else {
                params.set(LANDSCAPE_PARAM, value);
            }
            params.delete(VECTOR_DEPLOYMENT_PARAM);
            params.delete(DEEP_LINK_PARAM);
        });
    };

    const changeVectorDeployment = (value: string): void => {
        updateUrl((params) => {
            if (value === "") {
                params.delete(VECTOR_DEPLOYMENT_PARAM);
            } else {
                params.set(VECTOR_DEPLOYMENT_PARAM, value);
            }
            params.delete(DEEP_LINK_PARAM);
        });
    };

    const clearFilters = (): void => {
        query = "";
        status = "";
        if (serverFiltersActive || page.url.searchParams.has(DEEP_LINK_PARAM)) {
            updateUrl((params) => {
                params.delete(LANDSCAPE_PARAM);
                params.delete(VECTOR_DEPLOYMENT_PARAM);
                params.delete(DEEP_LINK_PARAM);
            });
        }
    };

    // Strip a stale ?deployment= that no longer resolves to a row.
    $effect(() => {
        if (!loading && rows.length > 0 && selectedId && selected === undefined) {
            closePanel();
        }
    });

    const selectClass =
        "min-w-[12rem] appearance-none rounded-[var(--input-radius)] border border-[color:var(--input-bd)] bg-[color:var(--input-bg)] px-3 py-2 text-[length:var(--text-sm)] text-[color:var(--input-fg)] focus-visible:border-[color:var(--border-strong)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50";
    const labelTextClass =
        "text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]";
</script>

<section
    class="mx-auto flex w-full max-w-[84rem] flex-col gap-5 px-6 pt-6 pb-10"
    aria-labelledby="artifact-view-title"
>
    <header class="flex flex-col gap-1">
        <h1
            id="artifact-view-title"
            class="m-0 text-[length:var(--text-h1)] font-[weight:var(--weight-display)] tracking-[var(--tracking-h1)] text-[color:var(--text-primary)]"
            data-testid="page-heading"
        >
            Artifact Deployments
        </h1>
        <p class="m-0 text-[length:var(--text-sm)] text-[color:var(--text-secondary)]">
            Inspect the artifact deployments backing this project's vectors, stages, and landscapes.
        </p>
    </header>

    {#if hasLoaded && !error}
        <div
            class="flex flex-wrap items-end gap-3 rounded-[var(--card-radius)] border border-[color:var(--border-subtle)] bg-[color:var(--surface-subtle)] px-4 py-3"
            data-testid="artifact-view-filters"
        >
            <div class="flex min-w-[16rem] grow-[2] basis-[20rem] flex-col gap-1">
                <span class={labelTextClass}>Search</span>
                <SearchInput
                    bind:value={query}
                    placeholder="Search artifact, version, stage, vector…"
                    aria-label="Search artifact deployments"
                    data-testid="artifact-search"
                    class="w-full max-w-none"
                />
            </div>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Status</span>
                <select
                    class={selectClass}
                    bind:value={status}
                    aria-label="Filter by status"
                    data-testid="artifact-status-filter"
                >
                    <option value="">All statuses</option>
                    <option value="ArtifactDeployed">Deployed</option>
                    <option value="ArtifactFetched">Fetched</option>
                </select>
            </label>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Landscape</span>
                <select
                    class={selectClass}
                    value={selectedLandscapeId ?? ""}
                    onchange={(event) => changeLandscape(event.currentTarget.value)}
                    disabled={landscapes.length === 0}
                    data-testid="artifact-landscape-filter"
                >
                    <option value="">All landscapes</option>
                    {#each landscapes as landscape (landscape.id)}
                        <option value={landscape.id}>{landscape.name}</option>
                    {/each}
                </select>
            </label>
            <label class="flex min-w-[12rem] flex-col gap-1">
                <span class={labelTextClass}>Vector deployment</span>
                <select
                    class={selectClass}
                    value={selectedVectorDeploymentId ?? ""}
                    onchange={(event) => changeVectorDeployment(event.currentTarget.value)}
                    disabled={vectorOptions.length === 0}
                    data-testid="artifact-vector-filter"
                >
                    <option value="">All vector deployments</option>
                    {#each vectorOptions as option (option.id)}
                        <option value={option.id}>{option.label} · {option.id}</option>
                    {/each}
                </select>
            </label>
            <div class="flex min-w-0 flex-col">
                <span class={labelTextClass} aria-hidden="true">&nbsp;</span>
                <button
                    type="button"
                    class="cursor-pointer border border-transparent bg-transparent px-1.5 py-2 text-[length:var(--text-sm)] leading-[1.4] text-[color:var(--text-link,var(--btn-primary-fg))] underline underline-offset-[3px] focus-visible:rounded-[var(--radius-sm)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none {anyFiltersActive
                        ? ''
                        : 'invisible pointer-events-none'}"
                    onclick={clearFilters}
                    disabled={!anyFiltersActive}
                    aria-hidden={!anyFiltersActive}
                    tabindex={anyFiltersActive ? 0 : -1}
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
                title={serverFiltersActive
                    ? "No artifact deployments match the current filters"
                    : "No artifact deployments"}
                description={serverFiltersActive
                    ? "Try widening the landscape or vector-deployment filter."
                    : "This project does not have any artifact deployments yet."}
            >
                {#snippet icon()}
                    <ui5-icon name="product" aria-hidden="true"></ui5-icon>
                {/snippet}
                {#snippet action()}
                    {#if serverFiltersActive}
                        <Button variant="secondary" onclick={clearFilters} data-testid="artifact-view-empty-clear">
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
                {visibleRows.length.toLocaleString()} of {rows.length.toLocaleString()} deployment{rows.length === 1 ? "" : "s"}
            </p>
            {#if visibleRows.length === 0}
                <div
                    class="flex flex-col items-center gap-2 rounded-[var(--card-radius)] border border-dashed border-[color:var(--border-default)] px-6 py-12 text-center text-[color:var(--text-secondary)]"
                    role="status"
                    data-testid="artifact-table-no-results"
                >
                    <p class="m-0">No artifact deployments match the current filters.</p>
                    <button
                        type="button"
                        class="cursor-pointer border-0 bg-transparent text-[color:var(--text-link,var(--btn-primary-fg))] underline underline-offset-[3px] focus-visible:rounded-[var(--radius-sm)] focus-visible:shadow-[var(--focus-ring)] focus-visible:outline-none"
                        onclick={clearFilters}
                    >
                        Clear filters
                    </button>
                </div>
            {:else}
                <ArtifactDeploymentsTable rows={visibleRows} {selectedId} onSelect={openRow} />
            {/if}
        </div>
    {/if}
</section>

<SidePanel
    open={panelOpen}
    title={selected ? `${selected.component}@${selected.version}` : "Artifact deployment"}
    onClose={closePanel}
>
    {#if selected}
        <ArtifactDeploymentDetail row={selected} />
    {/if}
</SidePanel>
