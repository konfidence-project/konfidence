<script lang="ts">
    import { Button, ExpandableGroup, SelectableRow } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let selectedSingle = $state<string | undefined>("first");
    let selectedMultiple = $state<string | undefined>("first");
    let selectedHeader = $state<string>();
    let headerExpanded = $state(false);
    let actionCount = $state(0);
</script>

<Section id="lists" title="Lists & rows" subtitle="Flexible row content with selection, expansion and independent actions." status="partial">
    <div class="mb-5 grid items-start gap-4 xl:grid-cols-2">
        <Sample title="Single-line rows" description="Free row content, with an optional independent action.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Single-line items" expanded bind:selectedValue={selectedSingle}>
                    {#snippet header()}<div class="font-semibold">Items</div>{/snippet}
                    <SelectableRow ariaLabel="Select first single-line item" value="first">
                        <span class="font-semibold">First item</span>
                        {#snippet actions()}<Button variant="secondary" onclick={() => actionCount += 1}>Inspect</Button>{/snippet}
                    </SelectableRow>
                    <SelectableRow ariaLabel="Select second single-line item" value="second"><span class="font-semibold">Second item</span></SelectableRow>
                </ExpandableGroup>
                <p class="mt-3 text-meta text-content-secondary" aria-live="polite">Selected: {selectedSingle || "none"}. Actions: {actionCount}.</p>
            </div>
        </Sample>
        <Sample title="Multiple-line rows" description="Parent-bound selection shared across groups. Long text wraps.">
            <div class="flex w-full min-w-0 flex-col gap-3">
                <ExpandableGroup ariaLabel="Multiple-line items" expanded bind:selectedValue={selectedMultiple}>
                    {#snippet header()}<div class="font-semibold">Items</div><div class="mt-1 text-content-secondary">Additional group information</div>{/snippet}
                    <SelectableRow ariaLabel="Select first multiple-line item" value="first">
                        <span class="flex min-w-0 items-center gap-3">
                            <span class="min-w-0 grow"><span class="block font-semibold">First item</span><span class="mt-1 block text-content-secondary">Additional information</span></span>
                            <span class="shrink-0 text-content-secondary">Details</span>
                        </span>
                    </SelectableRow>
                    <SelectableRow ariaLabel="Select second multiple-line item" value="second">
                        <span class="block font-semibold">An item with a longer title that can wrap naturally across multiple lines when space is limited</span>
                        <span class="mt-1 block text-content-secondary">Secondary text can also wrap across as many lines as its content needs.</span>
                    </SelectableRow>
                </ExpandableGroup>
                <ExpandableGroup ariaLabel="Other items" bind:selectedValue={selectedMultiple}>
                    {#snippet header()}<div class="font-semibold">Other items</div>{/snippet}
                    <SelectableRow ariaLabel="Select other item" value="other"><span class="font-semibold">Another item</span></SelectableRow>
                </ExpandableGroup>
                <p class="m-0 text-meta text-content-secondary" aria-live="polite">Selected: {selectedMultiple || "none"}.</p>
                <Button variant="ghost" onclick={() => selectedMultiple = undefined}>Clear selection</Button>
            </div>
        </Sample>
        <Sample title="Interactive header" description="An explicit shortcut opens the group and selects a row. Independent actions stop propagation.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Items with header action" bind:expanded={headerExpanded} bind:selectedValue={selectedHeader}>
                    {#snippet header()}
                        <div class="flex flex-wrap items-center justify-between gap-2">
                            <div class="font-semibold">Items</div>
                            <Button variant="secondary" onclick={(event: MouseEvent) => { event.stopPropagation(); headerExpanded = true; selectedHeader = "current"; }}>Show current</Button>
                        </div>
                    {/snippet}
                    <SelectableRow ariaLabel="Select current item" value="current" inset><span class="font-semibold">Current item</span></SelectableRow>
                    <SelectableRow ariaLabel="Select additional item" value="additional" inset>
                        <span class="block font-semibold">Additional item</span><span class="mt-1 block text-content-secondary">Visible when expanded</span>
                        {#snippet actions()}<Button variant="secondary" disabled>Unavailable action</Button>{/snippet}
                    </SelectableRow>
                </ExpandableGroup>
                <p class="mt-3 text-meta text-content-secondary" aria-live="polite">Selected: {selectedHeader || "none"}.</p>
            </div>
        </Sample>
        <Sample title="Local selection" description="Without a binding, the group manages its own selection.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Local items" expanded>
                    {#snippet header()}<div class="font-semibold">Local items</div>{/snippet}
                    <SelectableRow ariaLabel="Select local item" value="local">Local item</SelectableRow>
                </ExpandableGroup>
            </div>
        </Sample>
        <Sample title="Static group" description="Without rows, the header has no disclosure control.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Static items">
                    {#snippet header()}<div class="font-semibold">Items</div><div class="mt-1 text-content-secondary">No items available</div>{/snippet}
                </ExpandableGroup>
            </div>
        </Sample>
    </div>
    <NotYetImplemented planned={["<ActivityRow>", "<FilterCard>"]} />
</Section>
