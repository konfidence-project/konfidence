<script lang="ts">
    import SidePanel from "../SidePanel.svelte";

    interface Props {
        open?: boolean;
        title?: string;
        onClose?: () => void;
        showBody?: boolean;
        width?: number;
        onWidthChange?: (width: number) => void;
        headerHeight?: string;
    }
    let {
        open = $bindable(false),
        title = "Details",
        onClose,
        showBody = true,
        width,
        onWidthChange,
        headerHeight,
    }: Props = $props();

    // svelte-ignore state_referenced_locally
    let panelWidth = $state(width);
</script>

<div style:--app-shell-header-height={headerHeight} data-testid="fixture-root">
    <button type="button" data-testid="open-btn" onclick={() => (open = true)}>Open</button>
    {#if width === undefined}
        <SidePanel bind:open {title} {onClose} {onWidthChange}>
            {@render body()}
            {#snippet footer()}
                {@render footerContent()}
            {/snippet}
        </SidePanel>
    {:else}
        <SidePanel bind:open bind:width={panelWidth} {title} {onClose} {onWidthChange}>
            {@render body()}
            {#snippet footer()}
                {@render footerContent()}
            {/snippet}
        </SidePanel>
    {/if}
</div>

{#snippet body()}
    {#if showBody}
        <div style="display: flex; flex-direction: column; gap: 12px;">
            <p data-testid="panel-body" style="margin: 0; font-size: var(--font-size-compact); color: var(--text-primary);">
                Inspect the artifact and its landscape / stage / vector relationships.
            </p>
            <dl
                style="display: grid; grid-template-columns: 8rem 1fr; gap: 8px 12px; margin: 0; font-size: var(--font-size-compact);"
            >
                <dt style="color: var(--text-tertiary);">Component</dt>
                <dd style="margin: 0;">payments-api</dd>
                <dt style="color: var(--text-tertiary);">Version</dt>
                <dd style="margin: 0; font-family: var(--font-mono);">3.4.1</dd>
                <dt style="color: var(--text-tertiary);">Landscape</dt>
                <dd style="margin: 0;">Development</dd>
            </dl>
        </div>
    {/if}
{/snippet}

{#snippet footerContent()}
    <span data-testid="panel-footer" style="font-size: var(--font-size-compact); color: var(--text-tertiary);"
        >footer</span
    >
{/snippet}
