<script lang="ts">
    import TreeGroup from "../TreeGroup.svelte";
    import TreeRow from "../TreeRow.svelte";

    let { onOpenChange, onValueChange }: {
        onOpenChange?: (open: boolean) => void;
        onValueChange?: (value: string) => void;
    } = $props();
    let open = $state(true);
    let selectedValue = $state<string | undefined>("package");
</script>

<TreeGroup label="Changed Files" bind:open bind:selectedValue {onOpenChange} {onValueChange}>
    {#snippet metadata()}3 files{/snippet}
    <TreeRow value="index" label="src/index.ts">
        {#snippet metadata()}modified{/snippet}
        {#snippet trailing()}TypeScript{/snippet}
    </TreeRow>
    <TreeRow value="package" label="package.json" />
    <TreeRow value="disabled" label="Unavailable file" disabled />
</TreeGroup>
<TreeGroup label="Other group" open>
    <TreeRow value="index" label="Other file" />
</TreeGroup>
<output data-testid="selection">{selectedValue ?? "none"}</output>
<output data-testid="open">{String(open)}</output>
<button onclick={() => { selectedValue = undefined; open = false; }}>Reset externally</button>
