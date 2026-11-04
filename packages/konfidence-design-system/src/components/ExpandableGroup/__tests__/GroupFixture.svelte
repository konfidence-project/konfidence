<script lang="ts">
    import { Button, ExpandableGroup, SelectableRow } from "../../index.js";

    interface Props { extra?: boolean; inset?: boolean; withSummary?: boolean; title?: string; onselect?: () => void; onaction?: () => void; }
    let { extra = true, inset = false, withSummary = true, title = "Example group", onselect = () => {}, onaction = () => {} }: Props = $props();
    let expanded = $state(false);
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
    <ExpandableGroup ariaLabel="Example group" {expanded} onExpandedChange={(value) => expanded = value} children={extra ? history : undefined} summary={withSummary ? summaryContent : undefined}>
        {#snippet header()}<span>{title}</span>{/snippet}
    </ExpandableGroup>
</div>
