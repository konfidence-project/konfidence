<script lang="ts">
    import {
        StatusBadge,
        Table,
        TableCell,
        TableHeaderCell,
        TableRow,
    } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    interface DemoDeployment {
        id: string;
        artifact: string;
        version: string;
        stage: string;
        status: "healthy" | "warning" | "degraded" | "deploying" | "queued" | "error";
        touchedBy: string;
    }

    const DEPLOYMENTS: readonly DemoDeployment[] = [
        {
            artifact: "kden-cli",
            id: "d-001",
            stage: "prod",
            status: "healthy",
            touchedBy: "kaya@example.com",
            version: "v1.4.2",
        },
        {
            artifact: "kden-api",
            id: "d-002",
            stage: "staging",
            status: "deploying",
            touchedBy: "mike@example.com",
            version: "v1.5.0-rc.1",
        },
        {
            artifact: "kden-ui",
            id: "d-003",
            stage: "prod",
            status: "warning",
            touchedBy: "casey@example.com",
            version: "v0.9.7",
        },
        {
            artifact: "kden-mock-api",
            id: "d-004",
            stage: "dev",
            status: "queued",
            touchedBy: "alex@example.com",
            version: "v0.2.1",
        },
    ];

    let selectedId = $state<string | undefined>(undefined);

    type SortKey = "artifact" | "version" | "stage";
    let sortKey = $state<SortKey>("artifact");
    let sortDir = $state<"ascending" | "descending">("ascending");

    const sorted = $derived.by(() => {
        const rows = [...DEPLOYMENTS];
        rows.sort((left, right) => {
            const cmp = left[sortKey].localeCompare(right[sortKey]);
            return sortDir === "ascending" ? cmp : -cmp;
        });
        return rows;
    });

    const cycle = (key: SortKey): void => {
        if (sortKey !== key) {
            sortKey = key;
            sortDir = "ascending";
        } else {
            sortDir = sortDir === "ascending" ? "descending" : "ascending";
        }
    };

    const sortState = (key: SortKey): "ascending" | "descending" | "none" =>
        sortKey === key ? sortDir : "none";
</script>

<Section
    id="data-blocks"
    title="Data blocks"
    subtitle="Real <Table> family from @konfidence/design-system — sortable headers, keyboard-selectable rows, sticky header, embedded <StatusBadge>. KPI tiles, phases, timelines, diff rows are still on the roadmap."
    status="partial"
>
    <div class="stack">
        <Sample
            title="Sortable, selectable table"
            description="Click any row to select it; click a header to cycle its sort. Uses <TableHeaderCell onsort>, <TableRow onselect>, and <StatusBadge> for the state column."
        >
            <div class="table-wrap">
                <Table caption="Artifact deployments">
                    {#snippet header()}
                        <TableRow>
                            <TableHeaderCell
                                sort={sortState("artifact")}
                                onsort={() => cycle("artifact")}
                            >Artifact</TableHeaderCell>
                            <TableHeaderCell
                                sort={sortState("version")}
                                onsort={() => cycle("version")}
                            >Version</TableHeaderCell>
                            <TableHeaderCell
                                sort={sortState("stage")}
                                onsort={() => cycle("stage")}
                            >Stage</TableHeaderCell>
                            <TableHeaderCell>Status</TableHeaderCell>
                            <TableHeaderCell>Touched by</TableHeaderCell>
                        </TableRow>
                    {/snippet}
                    {#snippet body()}
                        {#each sorted as row (row.id)}
                            <TableRow
                                onselect={() => (selectedId = row.id)}
                                selected={row.id === selectedId}
                            >
                                <TableCell>{row.artifact}</TableCell>
                                <TableCell>{row.version}</TableCell>
                                <TableCell>{row.stage}</TableCell>
                                <TableCell>
                                    <StatusBadge status={row.status}>{row.status}</StatusBadge>
                                </TableCell>
                                <TableCell>{row.touchedBy}</TableCell>
                            </TableRow>
                        {/each}
                    {/snippet}
                </Table>
            </div>
            {#if selectedId}
                <p class="log">Selected: <b>{selectedId}</b></p>
            {/if}
        </Sample>

        <NotYetImplemented
            note="The reference design system ships these as composable atoms. Konfidence has not yet promoted them into the Svelte component surface."
            planned={[
                "<KpiTile variant='standard' | 'active' | 'gradient'>",
                "<StageCard>",
                "<Phases done active failed pending>",
                "<DiffRow added|removed|unchanged>",
                "<Timeline> / <TimelineItem>",
                "<KeyValue>",
            ]}
        />
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .table-wrap {
        width: 100%;
    }

    .log {
        margin: 0;
        font-family: var(--font-mono);
        font-size: var(--text-meta);
        color: var(--text-secondary);
        padding: 3px 8px;
        border-radius: 6px;
        background: var(--surface-subtle);
    }
</style>
