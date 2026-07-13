<script lang="ts">
    import "@ui5/webcomponents-icons/dist/sort-ascending.js";
    import "@ui5/webcomponents-icons/dist/sort-descending.js";
    import "@ui5/webcomponents-icons/dist/sort.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import { Render, Subscribe } from "@humanspeak/svelte-headless-table";
    import { type ArtifactSummary } from "$lib/artifacts";
    import createArtifactTable from "./columns";
    import { toStore } from "svelte/store";

    const { artifacts }: { artifacts: ArtifactSummary[] } = $props();

    const { table, columns } = createArtifactTable(toStore(() => artifacts));
    const {
        headerRows,
        pageRows,
        pluginStates,
        tableAttrs,
        tableBodyAttrs,
        visibleColumns,
    } = table.createViewModel(columns, { rowDataId: (row) => row.displayName });
    const { filterValue } = pluginStates.filter;
    const { filterValues } = pluginStates.colFilter;
    const {
        bottomSpacerHeight,
        measureRowAction,
        renderedRows,
        topSpacerHeight,
        totalRows,
        virtualScroll,
    } = pluginStates.virtual;

    const updateColumnFilter = (columnId: string, value: string): void => {
        const columnFilterValue = value || undefined;
        filterValues.update((current) => ({
            ...current,
            [columnId]: columnFilterValue,
        }));
    };

    const sortIcon = (order: "asc" | "desc" | undefined): string => {
        if (order === "asc") {
            return "sort-ascending";
        }
        if (order === "desc") {
            return "sort-descending";
        }
        return "sort";
    };
</script>

<div class="table-shell">
    <div class="toolbar">
        <label>
            <span>Search artifacts</span>
            <input
                type="search"
                placeholder="Name, version, manifest…"
                bind:value={$filterValue}
            />
        </label>
        <p aria-live="polite">
            {$totalRows.toLocaleString()} matches · {$renderedRows.toLocaleString()}
            rows rendered
        </p>
    </div>

    <div class="table-container" use:virtualScroll>
        <table {...$tableAttrs}>
            <thead>
                {#each $headerRows as headerRow (headerRow.id)}
                    <Subscribe attrs={headerRow.attrs()} let:attrs>
                        <tr {...attrs}>
                            {#each headerRow.cells as cell (cell.id)}
                                <Subscribe
                                    attrs={cell.attrs()}
                                    props={cell.props()}
                                    let:attrs
                                    let:props
                                >
                                    <th {...attrs}>
                                        <button
                                            class="sort-button"
                                            type="button"
                                            data-sort-order={props.sort.order ??
                                                "none"}
                                            onclick={props.sort.toggle}
                                        >
                                            <Render of={cell.render()} />
                                            <ui5-icon
                                                name={sortIcon(
                                                    props.sort.order,
                                                )}
                                                aria-hidden="true"
                                            ></ui5-icon>
                                        </button>
                                        {#if cell.id === "reuse" || cell.id === "status"}
                                            <select
                                                aria-label={`Filter ${cell.id}`}
                                                value={String(
                                                    $filterValues[cell.id] ??
                                                        "",
                                                )}
                                                onchange={(event) =>
                                                    updateColumnFilter(
                                                        cell.id,
                                                        event.currentTarget
                                                            .value,
                                                    )}
                                            >
                                                <option value="">All</option>
                                                {#if cell.id === "reuse"}
                                                    <option value="Yes"
                                                        >Yes</option
                                                    >
                                                    <option value="No"
                                                        >No</option
                                                    >
                                                {:else}
                                                    <option value="Ready"
                                                        >Ready</option
                                                    >
                                                    <option value="Not ready"
                                                        >Not ready</option
                                                    >
                                                    <option value="Unknown"
                                                        >Unknown</option
                                                    >
                                                {/if}
                                            </select>
                                        {:else}
                                            <input
                                                aria-label={`Filter ${cell.id}`}
                                                placeholder="Filter…"
                                                value={String(
                                                    $filterValues[cell.id] ??
                                                        "",
                                                )}
                                                oninput={(event) =>
                                                    updateColumnFilter(
                                                        cell.id,
                                                        event.currentTarget
                                                            .value,
                                                    )}
                                            />
                                        {/if}
                                    </th>
                                </Subscribe>
                            {/each}
                        </tr>
                    </Subscribe>
                {/each}
            </thead>
            <tbody {...$tableBodyAttrs}>
                {#if $topSpacerHeight > 0}
                    <tr aria-hidden="true">
                        <td
                            colspan={$visibleColumns.length}
                            style:height={`${$topSpacerHeight}px`}
                            class="spacer"
                        ></td>
                    </tr>
                {/if}
                {#each $pageRows as row (row.id)}
                    <Subscribe attrs={row.attrs()} let:attrs>
                        <tr {...attrs} use:measureRowAction={row.id}>
                            {#each row.cells as cell (cell.id)}
                                <Subscribe attrs={cell.attrs()} let:attrs>
                                    <td {...attrs}
                                        ><Render of={cell.render()} /></td
                                    >
                                </Subscribe>
                            {/each}
                        </tr>
                    </Subscribe>
                {/each}
                {#if $bottomSpacerHeight > 0}
                    <tr aria-hidden="true">
                        <td
                            colspan={$visibleColumns.length}
                            style:height={`${$bottomSpacerHeight}px`}
                            class="spacer"
                        ></td>
                    </tr>
                {/if}
            </tbody>
        </table>
    </div>
</div>

<style>
    .table-shell {
        overflow: hidden;
        border: 1px solid var(--sapGroup_ContentBorderColor);
        border-radius: var(--sapElement_BorderCornerRadius);
        background: var(--sapGroup_ContentBackground);
    }

    .toolbar {
        display: flex;
        align-items: end;
        justify-content: space-between;
        gap: 1rem;
        padding: 1rem;
        border-bottom: 1px solid var(--sapGroup_ContentBorderColor);
    }

    .toolbar label {
        display: grid;
        gap: 0.375rem;
        width: min(24rem, 100%);
        color: var(--sapContent_LabelColor);
        font-size: 0.75rem;
        font-weight: 600;
    }

    .toolbar p {
        margin: 0;
        color: var(--sapContent_LabelColor);
        font-size: 0.875rem;
        white-space: nowrap;
    }

    .table-container {
        height: min(65vh, 45rem);
        min-width: 0;
        overflow: auto;
    }

    table {
        width: 100%;
        border-collapse: collapse;
        table-layout: fixed;
    }

    thead {
        position: sticky;
        z-index: 1;
        top: 0;
        background: var(--sapList_HeaderBackground);
        box-shadow: 0 1px var(--sapGroup_ContentBorderColor);
    }

    th,
    td {
        height: 3rem;
        box-sizing: border-box;
        padding: 0.5rem 0.75rem;
        overflow: hidden;
        border-bottom: 1px solid var(--sapList_BorderColor);
        text-align: left;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    th {
        height: auto;
        vertical-align: top;
    }

    .sort-button {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        margin-bottom: 0.375rem;
        padding: 0;
        border: 0;
        background: transparent;
        color: var(--sapList_HeaderTextColor);
        font: inherit;
        font-weight: 700;
        cursor: pointer;
    }

    .sort-button ui5-icon {
        width: 1rem;
        height: 1rem;
        color: var(--sapContent_IconColor);
    }

    input,
    select {
        width: 100%;
        height: 2rem;
        box-sizing: border-box;
        padding: 0 0.5rem;
        border: 1px solid var(--sapField_BorderColor);
        border-radius: var(--sapField_BorderCornerRadius);
        background: var(--sapField_BackgroundStyle);
        color: var(--sapField_TextColor);
        font: inherit;
    }

    tbody tr:not([aria-hidden="true"]):hover {
        background: var(--sapList_Hover_Background);
    }

    .spacer {
        padding: 0;
        border: 0;
    }

    @media (max-width: 48rem) {
        .toolbar {
            align-items: stretch;
            flex-direction: column;
        }

        .toolbar p {
            white-space: normal;
        }

        table {
            width: 60rem;
        }
    }
</style>
