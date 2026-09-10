<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="forms"
    title="Forms & switches"
    subtitle="Inputs, checkbox, radio, switch variants."
    status="placeholder"
>
    <NotYetImplemented
        note="Skeleton v5 provides the underlying primitives (Switch, SegmentedControl, form controls) — Konfidence has not yet published its opinionated wrappers with amber focus glow and warm-tinted validation states."
        planned={[
            "<TextInput>",
            "<Checkbox>",
            "<Radio> / <RadioGroup>",
            "<Switch variant='amber' | 'onoff' | 'status' | 'segment'>",
            "Amber focus glow across form controls",
        ]}
    />
</Section>
