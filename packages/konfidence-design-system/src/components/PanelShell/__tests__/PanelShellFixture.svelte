<script lang="ts">
    import PanelShell from "../PanelShell.svelte";

    let {
        open = $bindable(false),
        modal = $bindable(false),
        onClose,
        secondOpen = $bindable(false),
    }: {
        open?: boolean;
        modal?: boolean;
        onClose?: () => void;
        secondOpen?: boolean;
    } = $props();
    let focusKey = $state("first");
    let text = $state("");
</script>

<div style="--app-shell-header-height: calc(4px + 60px)">
    <button type="button" data-testid="open" onclick={() => (open = true)}>Open</button>
    <button type="button" data-testid="outside">Outside</button>
    <button type="button" data-testid="set-open" onclick={() => (open = !open)}>Toggle open</button>
        <PanelShell bind:open {modal} {onClose} {focusKey} title="Panel" testid="panel" dialogStyle="width: 400px">
            <label>Draft <input bind:value={text} /></label>
            <button type="button" data-testid="modality" onclick={() => (modal = !modal)}>Toggle modality</button>
            <button type="button" data-testid="refocus" onclick={() => (focusKey = text)}>Refocus</button>
        </PanelShell>
</div>

{#if secondOpen}
    <PanelShell bind:open={secondOpen} title="Second panel" modal={false} testid="second-panel" dialogStyle="width: 220px">
        <p>Independent panel</p>
    </PanelShell>
{/if}
