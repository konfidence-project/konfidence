<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="pipeline"
    title="Pipeline patterns"
    subtitle="Composed stage cards, promotion (from→to), 4 detail levels."
    status="placeholder"
>
    <NotYetImplemented
        note="Requires the Data-block primitives (StageCard, Phases) plus a composition layer."
        planned={[
            "<StageCard> with 4 detail levels",
            "<Promotion from to>",
            "<PipelineTimeline>",
        ]}
    />
</Section>
