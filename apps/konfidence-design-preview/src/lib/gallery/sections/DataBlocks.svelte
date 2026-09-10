<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="data-blocks"
    title="Data blocks"
    subtitle="KPI tiles, stage cards, phases progress, tables, diff rows, timeline, key-value."
    status="placeholder"
>
    <NotYetImplemented
        note="The reference design.system ships these as composable atoms. Konfidence has not yet promoted them into the Svelte component surface."
        planned={[
            "<KpiTile variant='standard' | 'active' | 'gradient'>",
            "<StageCard>",
            "<Phases done active failed pending>",
            "<DataTable> with the .tbl chrome",
            "<DiffRow added|removed|unchanged>",
            "<Timeline> / <TimelineItem>",
            "<KeyValue>",
        ]}
    />
</Section>
