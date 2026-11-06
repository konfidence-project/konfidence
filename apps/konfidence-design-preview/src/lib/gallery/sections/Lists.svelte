<script lang="ts">
    import { Button, ExpandableGroup, SelectableRow } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let selectedSingle = $state<string | undefined>("first");
    let selectedMultiple = $state<string | undefined>("first");
    let selectedOther = $state<string>();
    let selectedSummary = $state<string>();
    let summaryExpanded = $state(false);
    let actionCount = $state(0);
</script>

<Section
    id="lists"
    title="Lists & rows"
    subtitle="Flexible row content with selection, expansion and independent actions."
    status="partial"
>
    <div class="mb-5 grid items-start gap-4 xl:grid-cols-2">
        <Sample title="Single-line rows" description="Labels only, with an optional independent action.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Single-line items" label="Items" expanded bind:selectedValue={selectedSingle}>
                    <SelectableRow ariaLabel="Select first single-line item" value="first" label="First item">
                        {#snippet actions()}<Button variant="secondary" onclick={() => actionCount += 1}>Inspect</Button>{/snippet}
                    </SelectableRow>
                    <SelectableRow ariaLabel="Select second single-line item" value="second" label="Second item" />
                    <SelectableRow ariaLabel="Unavailable single-line item" value="unavailable" label="Unavailable item" disabled />
                </ExpandableGroup>
                <p class="mt-3 text-meta text-content-secondary" aria-live="polite">Selected: {selectedSingle || "none"}. Actions: {actionCount}.</p>
            </div>
        </Sample>
        <Sample title="Multiple-line rows" description="The same behavior, with secondary text and optional content on the right. Long text wraps.">
            <div class="flex w-full min-w-0 flex-col gap-3">
                <ExpandableGroup ariaLabel="Multiple-line items" label="Items" expanded bind:selectedValue={selectedMultiple}>
                    {#snippet metadata()}Additional group information{/snippet}
                    <SelectableRow ariaLabel="Select first multiple-line item" value="first" label="First item">
                        {#snippet metadata()}Additional information{/snippet}
                        {#snippet trailing()}<span class="text-content-secondary">Details</span>{/snippet}
                    </SelectableRow>
                    <SelectableRow ariaLabel="Select second multiple-line item" value="second" label="An item with a longer title that can wrap naturally across multiple lines when space is limited">
                        {#snippet metadata()}Secondary text can also wrap across as many lines as its content needs.{/snippet}
                    </SelectableRow>
                </ExpandableGroup>
                <ExpandableGroup ariaLabel="Other items" label="Other items" bind:selectedValue={selectedOther}>
                    <SelectableRow ariaLabel="Select other item" value="other" label="Another item" />
                </ExpandableGroup>
                <p class="m-0 text-meta text-content-secondary" aria-live="polite">Selected: {selectedMultiple || "none"}. Other group: {selectedOther || "none"}.</p>
                <Button variant="ghost" onclick={() => selectedMultiple = undefined}>Clear selection</Button>
            </div>
        </Sample>
        <Sample title="Always-visible row" description="The first row stays visible while the group is collapsed.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Items with summary" label="Items" bind:expanded={summaryExpanded} bind:selectedValue={selectedSummary}>
                    {#snippet summary()}
                        <SelectableRow ariaLabel="Select always-visible item" value="current" label="Always-visible item" inset>
                            {#snippet actions()}<Button variant="secondary" onclick={() => actionCount += 1}>Inspect</Button>{/snippet}
                        </SelectableRow>
                    {/snippet}
                    <SelectableRow ariaLabel="Select additional item" value="additional" label="Additional item" inset>
                        {#snippet metadata()}Visible when expanded{/snippet}
                        {#snippet actions()}<Button variant="secondary" disabled>Unavailable action</Button>{/snippet}
                    </SelectableRow>
                </ExpandableGroup>
                <p class="mt-3 text-meta text-content-secondary" aria-live="polite">Selected: {selectedSummary || "none"}.</p>
            </div>
        </Sample>
        <Sample title="Static group" description="Without additional rows, the header has no expansion trigger.">
            <div class="w-full min-w-0">
                <ExpandableGroup ariaLabel="Static items" label="Items">
                    {#snippet summary()}
                        <SelectableRow ariaLabel="Unavailable static item" label="Unavailable item" disabled>
                            {#snippet metadata()}Additional information{/snippet}
                        </SelectableRow>
                    {/snippet}
                </ExpandableGroup>
            </div>
        </Sample>
    </div>
    <NotYetImplemented planned={["<ActivityRow>", "<FilterCard>"]} />
</Section>
