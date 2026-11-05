<script lang="ts">
    import "@ui5/webcomponents-icons/dist/sort-ascending.js";
    import "@ui5/webcomponents-icons/dist/sort-descending.js";
    import "@ui5/webcomponents-icons/dist/sort.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import { Render } from "@humanspeak/svelte-headless-table";
    import {
        Table,
        TableCell,
        TableHeaderCell,
        TableRow,
    } from "@konfidence/design-system/components";
    import MonoCell from "#lib/components/table-cells/MonoCell.svelte";
    import StatusCell from "#lib/components/table-cells/StatusCell.svelte";
    import TagListCell from "#lib/components/table-cells/TagListCell.svelte";

    import type { ArtifactDeploymentRow } from "./deployments.js";
    import { statusLabel, statusTone } from "./deployments.js";
    import { createArtifactTable } from "./create-artifact-table.js";

    interface Props {
        rows: readonly ArtifactDeploymentRow[];
        selectedId?: string | undefined;
        onSelect: (row: ArtifactDeploymentRow) => void;
    }

    let { rows, selectedId, onSelect }: Props = $props();

    const { table, columns } = createArtifactTable(() => [...rows]);
    const vm = table.createViewModel(columns, {
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

<Table {...vm.current.tableAttrs}>
    {#snippet header()}
        {#each vm.current.headerRows as headerRow (headerRow.id)}
            <TableRow {...headerRow.current.attrs}>
                {#each headerRow.cells as cell (cell.id)}
                    <TableHeaderCell
                        {...cell.current.attrs}
                        sort={ariaSort(cell.current.props.sort.order)}
                        onsort={cell.current.props.sort.toggle}
                    >
                        <Render of={cell.render()} />
                        <ui5-icon
                            class="size-icon-sm text-content-tertiary"
                            name={sortIcon(cell.current.props.sort.order)}
                            aria-hidden="true"
                        ></ui5-icon>
                    </TableHeaderCell>
                {/each}
            </TableRow>
        {/each}
    {/snippet}
    {#snippet body()}
        {#each vm.current.pageRows as row (row.id)}
            {@const rowData = row.original}
            <TableRow
                {...row.current.attrs}
                data-testid="artifact-row"
                data-row-id={rowData.id}
                selected={rowData.id === selectedId}
                onselect={() => onSelect(rowData)}
            >
                {#each row.cells as cell (cell.id)}
                    <TableCell {...cell.current.attrs} data-column={cell.id}>
                        {#if cell.id === "status"}
                            <StatusCell
                                label={statusLabel(rowData.status)}
                                tone={statusTone(rowData.status)}
                            />
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
                {/each}
            </TableRow>
        {/each}
    {/snippet}
</Table>
