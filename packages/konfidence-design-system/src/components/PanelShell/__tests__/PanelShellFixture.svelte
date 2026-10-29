<script lang="ts">
    import PanelShell from "../PanelShell.svelte";

    let {
        open = $bindable(false),
        modal = $bindable(false),
        onClose,
        onCloseComplete,
        secondOpen = $bindable(false),
    }: {
        open?: boolean;
        modal?: boolean;
        onClose?: () => void;
        onCloseComplete?: () => void;
        secondOpen?: boolean;
    } = $props();
    let focusKey = $state("first");
    let text = $state("");
    let mounted = $state(true);
</script>

<div style="--app-shell-header-height: calc(4px + 60px)">
    <button type="button" data-testid="open" onclick={() => (open = true)}>Open</button>
    <button type="button" data-testid="outside">Outside</button>
    <button type="button" data-testid="set-open" onclick={() => (open = !open)}>Toggle open</button>
    <button type="button" data-testid="unmount" onclick={() => (mounted = false)}>Unmount</button>
    {#if mounted}
        <PanelShell bind:open {modal} {onClose} {onCloseComplete} {focusKey} title="Panel" placement="right" testid="panel" dialogStyle="width: 400px">
            <label>Draft <input bind:value={text} /></label>
            <button type="button" data-testid="modality" onclick={() => (modal = !modal)}>Toggle modality</button>
            <button type="button" data-testid="refocus" onclick={() => (focusKey = text)}>Refocus</button>
        </PanelShell>
    {/if}
</div>

{#if secondOpen}
    <PanelShell bind:open={secondOpen} title="Second panel" modal={false} placement="bottom" testid="second-panel" dialogStyle="height: 220px">
        <p>Independent panel</p>
    </PanelShell>
{/if}
