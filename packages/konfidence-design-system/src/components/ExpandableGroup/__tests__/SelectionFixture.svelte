<script lang="ts">
    import { Button, ExpandableGroup, SelectableRow } from "../../index.js";

    let selectedValue = $state<string | undefined>("first");
    let expansion = $state<Record<string, boolean>>({});
    let mounted = $state(true);
    let actionCount = $state(0);
</script>

<div data-testid="selection-fixture" class="bg-surface-default p-3">
    {#if mounted}
        <ExpandableGroup ariaLabel="Selectable files" bind:expanded={() => expansion.files ?? false, (value) => expansion.files = value} bind:selectedValue>
            {#snippet header()}<div class="font-semibold">Changed Files</div><div class="mt-1 text-content-secondary">2 files</div>{/snippet}
            <SelectableRow ariaLabel="First file" value="first">
                <span class="flex min-w-0 items-center gap-3">
                    <span class="min-w-0 grow"><span class="block font-semibold">package.json</span><span class="mt-1 block text-content-secondary">Modified</span></span>
                    <span class="shrink-0">JSON</span>
                </span>
                {#snippet actions()}<Button variant="secondary" onclick={() => actionCount += 1}>Inspect file</Button><Button disabled>Unavailable action</Button>{/snippet}
            </SelectableRow>
            <SelectableRow ariaLabel="Second file" value="second">
                <span class="flex min-w-0 items-center gap-3">
                    <span class="min-w-0 grow"><span class="block font-semibold">src/index.ts</span><span class="mt-1 block text-content-secondary">Modified</span></span>
                    <span class="shrink-0">TypeScript</span>
                </span>
            </SelectableRow>
        </ExpandableGroup>
        <ExpandableGroup ariaLabel="Other files" expanded bind:selectedValue={() => selectedValue, (value) => selectedValue = value}>
            {#snippet header()}<div class="font-semibold">Other Files</div>{/snippet}
            <SelectableRow ariaLabel="Other file" value="other"><span class="font-semibold">README.md</span></SelectableRow>
        </ExpandableGroup>
    {/if}
</div>
<!-- Keep test-control shadows outside the captured sample. -->
<div class="overflow-hidden p-3">
    <Button onclick={() => { selectedValue = undefined; expansion.files = false; }}>Reset externally</Button>
    <Button onclick={() => selectedValue = "second"}>Select externally</Button>
    <Button onclick={() => mounted = !mounted}>Toggle groups</Button>
    <span data-testid="selected-value">{selectedValue ?? "none"}</span>
    <span data-testid="action-count">{actionCount}</span>
</div>
