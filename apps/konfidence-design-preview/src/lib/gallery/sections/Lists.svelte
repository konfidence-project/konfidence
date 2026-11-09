<script lang="ts">
    import { Button, ExpandableGroup, SelectableRow } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";

    let expanded = $state(false);
    let headerOnlyExpanded = $state(false);
    let selected = $state("");
    let actionCount = $state(0);
</script>

<Section
    id="lists"
    title="Lists & rows"
    subtitle="Controlled expandable groups and selectable rows with independent actions."
    status="partial"
>
    <div class="mb-5 max-w-3xl">
        <ExpandableGroup ariaLabel="Example group" {expanded} onExpandedChange={(value) => expanded = value}>
            {#snippet header()}<span class="font-semibold">Example group</span>{/snippet}
            {#snippet summary()}
                <SelectableRow ariaLabel="Select current item" selected={selected === "current"} onselect={() => selected = "current"}>
                    <span class="flex flex-col gap-1"><span class="font-semibold">Always-visible item</span><span class="text-compact text-content-secondary">The summary remains visible while collapsed.</span></span>
                    {#snippet actions()}<Button variant="secondary" onclick={() => actionCount += 1}>Independent action</Button>{/snippet}
                </SelectableRow>
            {/snippet}
            <SelectableRow ariaLabel="Select older item" selected={selected === "older"} onselect={() => selected = "older"}>
                <span>Older item revealed on expansion</span>
                {#snippet actions()}<Button variant="secondary" disabled>Unavailable action</Button>{/snippet}
            </SelectableRow>
        </ExpandableGroup>
        <div class="mt-4">
            <ExpandableGroup ariaLabel="Header-only example" expanded={headerOnlyExpanded} onExpandedChange={(value) => headerOnlyExpanded = value}>
                {#snippet header()}<span class="font-semibold">Header-only collapsed group</span>{/snippet}
                <SelectableRow ariaLabel="Select hidden item" selected={selected === "hidden"} onselect={() => selected = "hidden"}>
                    <span>All rows are hidden until expanded, even with a single item.</span>
                </SelectableRow>
            </ExpandableGroup>
        </div>
        <p class="mt-3 text-compact text-content-secondary" aria-live="polite">Selected: {selected || "none"}. Independent actions: {actionCount}.</p>
    </div>
    <NotYetImplemented
        planned={["<ActivityRow>", "<FilterCard>"]}
    />
</Section>
