<script lang="ts">
    import { Button, ExpandableGroup, SelectableRow } from "../../index.js";

    interface Props {
        onValueChange?: (value: string) => void;
        onOpenChange?: (open: boolean) => void;
        onselect?: () => void;
        onaction?: () => void;
    }
    let { onValueChange, onOpenChange, onselect, onaction }: Props = $props();
    let selectedValue = $state<string | undefined>("first");
    let expanded = $state(true);
    let standaloneSelected = $state(false);
</script>

<div data-testid="selection-fixture" class="bg-surface-default p-3">
    <ExpandableGroup ariaLabel="Selectable files" label="Changed Files" bind:expanded bind:selectedValue {onValueChange} onExpandedChange={onOpenChange}>
        {#snippet metadata()}3 files{/snippet}
        <SelectableRow ariaLabel="First file" value="first" label="package.json" {onselect}>
            {#snippet metadata()}Modified{/snippet}
            {#snippet trailing()}JSON{/snippet}
            {#snippet actions()}<Button variant="secondary" onclick={onaction}>Inspect file</Button>{/snippet}
        </SelectableRow>
        <SelectableRow ariaLabel="Second file" value="second" label="src/index.ts" {onselect}>
            {#snippet metadata()}Modified{/snippet}
            {#snippet trailing()}TypeScript{/snippet}
        </SelectableRow>
        <SelectableRow ariaLabel="Disabled file" value="disabled" label="Unavailable file" disabled {onselect} />
    </ExpandableGroup>
    <ExpandableGroup ariaLabel="Other files" label="Other Files" expanded>
        <SelectableRow ariaLabel="Other file" value="other" label="README.md" />
    </ExpandableGroup>
    <SelectableRow ariaLabel="Standalone file" label="Standalone file" selected={standaloneSelected} onselect={() => standaloneSelected = !standaloneSelected} />
    <Button onclick={() => { selectedValue = undefined; expanded = false; }}>Reset externally</Button>
    <span data-testid="selected-value">{selectedValue ?? "none"}</span>
    <span data-testid="expanded-value">{String(expanded)}</span>
</div>
