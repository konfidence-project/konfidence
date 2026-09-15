<script lang="ts">
    import SidePanel from "../SidePanel.svelte";

    interface Props {
        open?: boolean;
        title?: string;
        onClose?: () => void;
        showBody?: boolean;
    }
    let {
        open = $bindable(false),
        title = "Details",
        onClose,
        showBody = true,
    }: Props = $props();
</script>

<button type="button" data-testid="open-btn" onclick={() => (open = true)}>Open</button>
<SidePanel bind:open {title} {onClose}>
    {#if showBody}
        <div style="display: flex; flex-direction: column; gap: 12px;">
            <p data-testid="panel-body" style="margin: 0; font-size: var(--text-sm); color: var(--text-primary);">
                Inspect the artifact and its landscape / stage / vector relationships.
            </p>
            <dl
                style="display: grid; grid-template-columns: 8rem 1fr; gap: 8px 12px; margin: 0; font-size: var(--text-sm);"
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
    {#snippet footer()}
        <span data-testid="panel-footer" style="font-size: var(--text-sm); color: var(--text-tertiary);"
            >footer</span
        >
    {/snippet}
</SidePanel>
