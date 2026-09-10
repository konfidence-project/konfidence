<script lang="ts">
    import Table from "../Table.svelte";
    import TableCell from "../TableCell.svelte";
    import TableHeaderCell from "../TableHeaderCell.svelte";
    import TableRow from "../TableRow.svelte";

    interface Props {
        caption?: string;
        onselect?: () => void;
    }
    let { caption, onselect }: Props = $props();
</script>

<div
    class="w-[540px] bg-[color:var(--surface-canvas)] p-4 text-[color:var(--text-primary)]"
    data-testid="table-frame"
>
    <Table {caption}>
        {#snippet header()}
            <TableRow>
                <TableHeaderCell>Deployment</TableHeaderCell>
                <TableHeaderCell>Artifact</TableHeaderCell>
                <TableHeaderCell>Version</TableHeaderCell>
            </TableRow>
        {/snippet}
        {#snippet body()}
            <TableRow data-testid="row-1" {onselect}>
                <TableCell>artifact-1</TableCell>
                <TableCell>payments-api</TableCell>
                <TableCell>3.4.1</TableCell>
            </TableRow>
            <TableRow data-testid="row-2" {onselect}>
                <TableCell>artifact-2</TableCell>
                <TableCell>payments-ui</TableCell>
                <TableCell>2.7.0</TableCell>
            </TableRow>
        {/snippet}
    </Table>
</div>
