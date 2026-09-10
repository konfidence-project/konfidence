<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="misc"
    title="Miscellaneous primitives"
    subtitle="Tooltip, dropdown menu, accordion, pagination."
    status="partial"
>
    <NotYetImplemented
        note="Skeleton v5 provides these primitives. Konfidence has not yet wrapped them with the design-system chrome. `<Menu.*>` is the only one already wrapped (see Shell patterns section)."
        planned={["<Tooltip>", "<Dropdown>", "<Accordion>", "<Pagination>"]}
    />
</Section>
