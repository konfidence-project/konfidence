<script lang="ts">
    import "@ui5/webcomponents-icons/dist/sort-ascending.js";
    import "@ui5/webcomponents-icons/dist/sort-descending.js";
    import "@ui5/webcomponents-icons/dist/sort.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import { Render, Subscribe } from "@humanspeak/svelte-headless-table";
    import {
        Table,
        TableCell,
        TableHeaderCell,
        TableRow,
    } from "@konfidence/design-system/components";
    import { toStore } from "svelte/store";

    import MonoCell from "./cells/MonoCell.svelte";
    import StatusCell from "./cells/StatusCell.svelte";
    import TagListCell from "./cells/TagListCell.svelte";
    import type { ArtifactDeploymentRow } from "./deployments.js";
    import { createArtifactTable } from "./createArtifactTable.js";

    interface Props {
        rows: readonly ArtifactDeploymentRow[];
        selectedId?: string | undefined;
        onSelect: (row: ArtifactDeploymentRow) => void;
    }

    let { rows, selectedId, onSelect }: Props = $props();

    const rowsStore = toStore(() => [...rows]);
    const { table, columns } = createArtifactTable(rowsStore);
    const { headerRows, pageRows, tableAttrs } = table.createViewModel(columns, {
        rowDataId: (row) => row.id,
    });

    const sortIcon = (order: "asc" | "desc" | undefined): string => {
        if (order === "asc") {
            return "sort-ascending";
        }
        if (order === "desc") {
            return "sort-descending";
        }
        return "sort";
    };

    const ariaSort = (order: "asc" | "desc" | undefined): "ascending" | "descending" | "none" => {
        if (order === "asc") {
            return "ascending";
        }
        if (order === "desc") {
            return "descending";
        }
        return "none";
    };
</script>

<Table {...$tableAttrs}>
    {#snippet header()}
        {#each $headerRows as headerRow (headerRow.id)}
            <Subscribe attrs={headerRow.attrs()} let:attrs>
                <TableRow {...attrs}>
                    {#each headerRow.cells as cell (cell.id)}
                        <Subscribe attrs={cell.attrs()} props={cell.props()} let:attrs let:props>
                            <TableHeaderCell
                                {...attrs}
                                sort={ariaSort(props.sort.order)}
                                onsort={props.sort.toggle}
                            >
                                <Render of={cell.render()} />
                                <ui5-icon
                                    class="h-[var(--icon-sm)] w-[var(--icon-sm)] text-[color:var(--text-tertiary)]"
                                    name={sortIcon(props.sort.order)}
                                    aria-hidden="true"
                                ></ui5-icon>
                            </TableHeaderCell>
                        </Subscribe>
                    {/each}
                </TableRow>
            </Subscribe>
        {/each}
    {/snippet}
    {#snippet body()}
        {#each $pageRows as row (row.id)}
            {@const rowData = row.original}
            <Subscribe attrs={row.attrs()} let:attrs>
                <TableRow
                    {...attrs}
                    data-testid="artifact-row"
                    data-row-id={rowData.id}
                    selected={rowData.id === selectedId}
                    onselect={() => onSelect(rowData)}
                >
                    {#each row.cells as cell (cell.id)}
                        <Subscribe attrs={cell.attrs()} let:attrs>
                            <TableCell {...attrs} data-column={cell.id}>
                                {#if cell.id === "status"}
                                    <StatusCell status={rowData.status} />
                                {:else if cell.id === "stages"}
                                    <TagListCell labels={rowData.stageNames} />
                                {:else if cell.id === "vectorDeployments"}
                                    <TagListCell
                                        labels={rowData.vectorDeploymentLabels}
                                        keys={rowData.vectorDeploymentIds}
                                        mono
                                    />
                                {:else if cell.id === "id" || cell.id === "version" || cell.id === "repository"}
                                    <MonoCell><Render of={cell.render()} /></MonoCell>
                                {:else}
                                    <Render of={cell.render()} />
                                {/if}
                            </TableCell>
                        </Subscribe>
                    {/each}
                </TableRow>
            </Subscribe>
        {/each}
    {/snippet}
</Table>
