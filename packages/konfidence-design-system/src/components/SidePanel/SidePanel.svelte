<script lang="ts">
    import type { Snippet } from "svelte";
    import "@ui5/webcomponents-icons/dist/resize.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import PanelShell from "../PanelShell/PanelShell.svelte";

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
    const KEYBOARD_STEP = 16;

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

    const clamp = (value: number): number => Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, Math.round(value)));

    const effectiveWidth = $derived(clamp(width));

    let resizing = $state(false);
    let dragStartX = 0;
    let dragStartWidth = 0;

    const commitWidth = (next: number): void => {
        const clamped = clamp(next);
        if (clamped !== width) {
            width = clamped;
        }
        onWidthChange?.(clamped);
    };

    const onHandlePointerDown = (event: PointerEvent): void => {
        if (event.button !== 0) {
            return;
        }
        event.preventDefault();
        resizing = true;
        dragStartX = event.clientX;
        dragStartWidth = effectiveWidth;
        (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    };

    const onHandlePointerMove = (event: PointerEvent): void => {
        if (!resizing) {
            return;
        }
        const delta = dragStartX - event.clientX;
        width = clamp(dragStartWidth + delta);
    };

    const onHandlePointerUp = (event: PointerEvent): void => {
        if (!resizing) {
            return;
        }
        resizing = false;
        (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
        onWidthChange?.(effectiveWidth);
    };

    const onHandleKeydown = (event: KeyboardEvent): void => {
        if (event.key === "ArrowLeft") {
            event.preventDefault();
            commitWidth(effectiveWidth + KEYBOARD_STEP);
        } else if (event.key === "ArrowRight") {
            event.preventDefault();
            commitWidth(effectiveWidth - KEYBOARD_STEP);
        } else if (event.key === "Home") {
            event.preventDefault();
            commitWidth(MAX_WIDTH);
        } else if (event.key === "End") {
            event.preventDefault();
            commitWidth(MIN_WIDTH);
        }
    };
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
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
            class="side-panel__resizer absolute inset-y-0 left-0 z-10 w-1.5 cursor-ew-resize touch-none bg-transparent transition-colors hover:bg-[var(--border-strong,rgba(120,90,40,0.5))] focus-visible:bg-[var(--border-strong,rgba(120,90,40,0.7))] focus-visible:outline-none"
            role="separator"
            aria-orientation="vertical"
            aria-label="Resize panel"
            aria-valuemin={MIN_WIDTH}
            aria-valuemax={MAX_WIDTH}
            aria-valuenow={effectiveWidth}
            tabindex="0"
            data-testid="side-panel-resizer"
            onpointerdown={onHandlePointerDown}
            onpointermove={onHandlePointerMove}
            onpointerup={onHandlePointerUp}
            onkeydown={onHandleKeydown}
        ></div>
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

        :global(.side-panel .side-panel__resizer) {
            display: none;
        }
    }
</style>
