<script lang="ts">
    import { untrack } from "svelte";
    import { Button, ExpandableGroup, SelectableRow } from "../../index.js";

    interface Props { initialExpanded?: boolean; extra?: boolean; inset?: boolean; title?: string; withHeaderAction?: boolean; }
    let { initialExpanded = false, extra = true, inset = false, title = "Example group", withHeaderAction = false }: Props = $props();
    let expanded = $state(untrack(() => initialExpanded));
    let actionCount = $state(0);
</script>
<div class="bg-surface-default p-3" data-testid="fixture">
    {#snippet history()}
        <SelectableRow ariaLabel="Select history" value="history" {inset}>
            <span class="block font-semibold">Historical item</span>
            {#snippet actions()}<Button variant="secondary" onclick={() => actionCount += 1}>Action</Button><Button disabled>Disabled</Button>{/snippet}
        </SelectableRow>
    {/snippet}
    <ExpandableGroup ariaLabel="Example group" triggerId={extra ? undefined : "example-header"} bind:expanded children={extra ? history : undefined}>
        {#snippet header()}
            <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="min-w-0 font-semibold">{title}</div>
                {#if withHeaderAction}<Button variant="secondary" onclick={(event: MouseEvent) => { event.stopPropagation(); actionCount += 1; }}>Header action</Button>{/if}
            </div>
        {/snippet}
    </ExpandableGroup>
</div>
<span data-testid="action-count">{actionCount}</span>
