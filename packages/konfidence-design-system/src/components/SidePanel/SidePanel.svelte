<script lang="ts">
    import type { Snippet } from "svelte";
    import "@ui5/webcomponents-icons/dist/resize.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import PanelShell from "../PanelShell/PanelShell.svelte";
    import PanelResizer from "../PanelResizer/PanelResizer.svelte";
    import { clampPanelSize } from "../PanelResizer/panel-size";

    interface Props {
        open?: boolean;
        title: string;
        children: Snippet;
        footer?: Snippet;
        onClose?: () => void;
        fullPageHref?: string;
        width?: number;
        onWidthChange?: (width: number) => void;
        modal?: boolean;
        focusKey?: string;
        onCloseComplete?: () => void;
    }

    const DEFAULT_WIDTH = 544;
    const MIN_WIDTH = 360;
    const MAX_WIDTH = 960;

    let {
        open = $bindable(false),
        title,
        children,
        footer,
        onClose,
        fullPageHref,
        width = $bindable(DEFAULT_WIDTH),
        onWidthChange,
        modal = true,
        focusKey,
        onCloseComplete,
    }: Props = $props();

    const effectiveWidth = $derived(clampPanelSize(width, MIN_WIDTH, MAX_WIDTH));
</script>

<PanelShell
    bind:open
    {title}
    placement="right"
    testid="side-panel"
    dialogClass="side-panel border-l border-l-outline-subtle shadow-[-8px_0_24px_rgba(0,0,0,0.12)] backdrop:bg-[linear-gradient(to_bottom,transparent_0,transparent_var(--app-shell-header-height,0px),var(--scrim)_var(--app-shell-header-height,0px))]"
    dialogStyle="width: min({effectiveWidth}px, 100vw)"
    {onClose}
    {modal}
    {focusKey}
    {onCloseComplete}
    {footer}
>
    {#snippet resizer()}
        <PanelResizer
            edge="left"
            bind:size={width}
            min={MIN_WIDTH}
            max={MAX_WIDTH}
            testid="side-panel-resizer"
            onResize={onWidthChange}
        />
    {/snippet}
    {#snippet headerAction()}
        {#if fullPageHref}
            <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- The design-system caller supplies an already-resolved application URL. -->
            <a
                href={fullPageHref}
                class="cursor-pointer rounded-[var(--radius-sm)] border-0 bg-transparent p-1.5 leading-none text-content-tertiary hover:bg-surface-sunken hover:text-content-primary focus-visible:outline-none focus-visible:shadow-[var(--focus-ring)] [&_ui5-icon]:size-icon-md"
                aria-label="Open full page"
                data-testid="side-panel-full-page"
            >
                <ui5-icon class="text-current" name="resize" aria-hidden="true"></ui5-icon>
            </a>
        {/if}
    {/snippet}
    {@render children()}
</PanelShell>

<style>
    @media (max-width: 767px) {
        :global(.side-panel) {
            width: 100vw !important;
        }

    }
</style>
