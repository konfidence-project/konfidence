<script lang="ts">
    import BottomSheet from "../BottomSheet.svelte";

    interface Props {
        open?: boolean;
        title?: string;
        onClose?: () => void;
        showBody?: boolean;
        overflow?: boolean;
        height?: number;
        onHeightChange?: (height: number) => void;
        headerHeight?: string;
        modal?: boolean;
        onCloseComplete?: () => void;
    }
    let {
        open = $bindable(false),
        title = "Details",
        onClose,
        showBody = true,
        overflow = false,
        height,
        onHeightChange,
        headerHeight,
        modal = true,
        onCloseComplete,
    }: Props = $props();

    // svelte-ignore state_referenced_locally
    let panelHeight = $state(height);
</script>

<div style:--app-shell-header-height={headerHeight} data-testid="fixture-root">
    <button type="button" data-testid="open-btn" onclick={() => (open = true)}>Open</button>
    <button type="button" data-testid="outside-btn">Outside action</button>
    {#if height === undefined}
        <BottomSheet bind:open {title} {onClose} {onCloseComplete} {onHeightChange} {modal}>
            {@render body()}
            {#snippet footer()}
                {@render footerContent()}
            {/snippet}
        </BottomSheet>
    {:else}
        <BottomSheet bind:open bind:height={panelHeight} {title} {onClose} {onCloseComplete} {onHeightChange} {modal}>
            {@render body()}
            {#snippet footer()}
                {@render footerContent()}
            {/snippet}
        </BottomSheet>
    {/if}
</div>

{#snippet body()}
    {#if showBody}
        <div style="display: flex; flex-direction: column; gap: 12px;">
            <p
                data-testid="panel-body"
                style="margin: 0; font-size: var(--font-size-compact); color: var(--text-primary);"
            >
                Review the promotion from source to target.
            </p>
            {#if overflow}
                <div data-testid="panel-overflow" style="height: 4000px;"></div>
            {/if}
        </div>
    {/if}
{/snippet}

{#snippet footerContent()}
    <span
        data-testid="panel-footer"
        style="font-size: var(--font-size-compact); color: var(--text-tertiary);">footer</span
    >
{/snippet}
