<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="charts"
    title="Diagrams & charts"
    subtitle="Bars, stacked bars, donut, line/area, flow, funnel, environment-health meter."
    status="placeholder"
>
    <NotYetImplemented
        note="The reference design ships these as SVG + CSS. Konfidence has not yet chosen a charting library (or reimplemented the SVG helpers in Svelte)."
        planned={[
            "<BarChart>",
            "<StackedBarChart>",
            "<LineChart> / <AreaChart>",
            "<Donut>",
            "<FlowDiagram>",
            "<Funnel>",
            "<HealthMeter>",
            "<Sparkline>",
        ]}
    />
</Section>
