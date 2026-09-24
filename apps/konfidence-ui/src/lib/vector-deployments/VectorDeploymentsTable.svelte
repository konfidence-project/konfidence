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
    import { resolve } from "$app/paths";
    import { VECTOR_DEPLOYMENT_PARAM } from "$lib/deployments/params";
    import LinkCell from "$lib/components/table-cells/LinkCell.svelte";
    import StatusCell from "$lib/components/table-cells/StatusCell.svelte";
    import { toStore } from "svelte/store";

    import { createVectordeploymentTable } from "./create-vectordeployment-table.js";
    import type { VectorDeploymentRow } from "./deployments.js";
    import { statusLabel, statusTone } from "./deployments.js";

    interface Props {
        projectId: string;
        rows: readonly VectorDeploymentRow[];
        selectedId?: string | undefined;
        onSelect: (row: VectorDeploymentRow) => void;
    }

    let { projectId, rows, selectedId, onSelect }: Props = $props();

    const rowsStore = toStore(() => [...rows]);
    const { table, columns } = createVectordeploymentTable(rowsStore);
    const { headerRows, pageRows, tableAttrs } = table.createViewModel(columns, {
        rowDataId: (row) => row.id,
    });

    const artifactDeploymentsUrl = (deploymentId: string) =>
        resolve(
            `/(shell)/projects/[projectId]/artifact-deployments?${VECTOR_DEPLOYMENT_PARAM}=${encodeURIComponent(deploymentId)}`,
            { projectId },
        );

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
                    data-testid="vectordeployment-row"
                    data-row-id={rowData.id}
                    selected={rowData.id === selectedId}
                    onselect={() => onSelect(rowData)}
                >
                    {#each row.cells as cell (cell.id)}
                        <Subscribe attrs={cell.attrs()} let:attrs>
                            <TableCell {...attrs} data-column={cell.id}>
                                {#if cell.id === "status"}
                                    <StatusCell
                                        label={statusLabel(rowData.status)}
                                        tone={statusTone(rowData.status)}
                                    />
                                {:else if cell.id === "relatedArtifactDeployments"}
                                    <LinkCell
                                        url={artifactDeploymentsUrl(rowData.id)}
                                        ariaLabel={`View artifact deployments for ${rowData.id}`}
                                    >
                                        {rowData.relatedArtifactDeployments.length}
                                    </LinkCell>
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
