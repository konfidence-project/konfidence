<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="foundation-atoms"
    title="Foundation · Atoms"
    subtitle=".status-dot, .live, .ver, .meta-row, .stage-line, .stat, .titleline — small building blocks that compose into stage cards and KPI tiles."
    status="placeholder"
>
    <NotYetImplemented
        note="The reference design ships these as tiny CSS-only atoms. Konfidence ships an equivalent for the shell (SkipLink, Brandbar) but not for the composable data atoms yet."
        planned={[
            "<StatusDot status>",
            "<LiveIndicator>",
            "<Version>",
            "<MetaRow>",
            "<StageLine>",
            "<Stat value label>",
            "<TitleLine>",
        ]}
    />
</Section>
