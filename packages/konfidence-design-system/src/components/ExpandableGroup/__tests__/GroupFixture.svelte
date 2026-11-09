<script lang="ts">
    import { untrack } from "svelte";
    import { Button, ExpandableGroup, SelectableRow } from "../../index.js";

    interface Props { initialExpanded?: boolean; acceptExpansion?: boolean; onExpandedChange?: (expanded: boolean) => void; extra?: boolean; inset?: boolean; withSummary?: boolean; title?: string; onselect?: () => void; onaction?: () => void; }
    let { initialExpanded = false, acceptExpansion = true, onExpandedChange, extra = true, inset = false, withSummary = true, title = "Example group", onselect = () => {}, onaction = () => {} }: Props = $props();
    let expanded = $state(untrack(() => initialExpanded));
    let selected = $state(false);
</script>
<div class="bg-surface-default p-3" data-testid="fixture">
    {#snippet history()}
        <SelectableRow ariaLabel="Select history" {inset} onselect={onselect}><span>Historical item</span></SelectableRow>
    {/snippet}
        {#snippet summaryContent()}
            <SelectableRow ariaLabel="Select summary" {selected} {inset} onselect={() => { selected = true; onselect(); }}>
                <span>Always visible summary</span>
                {#snippet actions()}<Button variant="secondary" onclick={onaction}>Action</Button><Button disabled>Disabled</Button>{/snippet}
            </SelectableRow>
        {/snippet}
    <ExpandableGroup ariaLabel="Example group" {expanded} onExpandedChange={(value) => { onExpandedChange?.(value); if (acceptExpansion) expanded = value; }} children={extra ? history : undefined} summary={withSummary ? summaryContent : undefined}>
        {#snippet header()}<span>{title}</span>{/snippet}
    </ExpandableGroup>
</div>
