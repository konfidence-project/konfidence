<script lang="ts">
    import Section from "../Section.svelte";
    import NotYetImplemented from "../NotYetImplemented.svelte";

    let { name = "Component" }: { name?: string } = $props();
</script>

<Section id={name} title={name} status="placeholder">
    <NotYetImplemented planned={[`<${name}>`]} />
</Section>
