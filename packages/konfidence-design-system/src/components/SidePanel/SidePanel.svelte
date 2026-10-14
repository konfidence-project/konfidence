<script lang="ts">
    import type { Snippet } from "svelte";
    import "@ui5/webcomponents-icons/dist/decline.js";
    import "@ui5/webcomponents/dist/Icon.js";

    interface Props {
        open?: boolean;
        title: string;
        children: Snippet;
        footer?: Snippet;
        onClose?: () => void;
        width?: number;
        onWidthChange?: (width: number) => void;
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
        width = $bindable(DEFAULT_WIDTH),
        onWidthChange,
    }: Props = $props();

    let dialog: HTMLDialogElement | undefined = $state();

    const clamp = (value: number): number => Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, Math.round(value)));

    const effectiveWidth = $derived(clamp(width));

    $effect(() => {
        const target = dialog;
        if (!target) {
            return;
        }
        if (open && !target.open) {
            target.showModal();
        } else if (!open && target.open) {
            target.close();
        }
    });

    const requestClose = (): void => {
        open = false;
        onClose?.();
    };

    const handleBackdropClick = (event: MouseEvent): void => {
        if (event.target === dialog) {
            requestClose();
        }
    };

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

<dialog
    bind:this={dialog}
    class="side-panel fixed top-[var(--app-shell-header-height,0px)] bottom-0 right-0 left-auto m-0 h-auto max-h-[calc(100dvh-var(--app-shell-header-height,0px))] max-w-[100vw] flex-col overflow-hidden border-0 border-l border-l-outline-subtle bg-surface-default p-0 text-content-primary shadow-[-8px_0_24px_rgba(0,0,0,0.12)] backdrop:bg-[linear-gradient(to_bottom,transparent_0,transparent_var(--app-shell-header-height,0px),var(--scrim)_var(--app-shell-header-height,0px))]"
    style:width="min({effectiveWidth}px, 100vw)"
    aria-label={title}
    data-testid="side-panel"
    onclose={requestClose}
    onclick={handleBackdropClick}
>
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
    <header
        class="flex items-center justify-between gap-2 border-b border-[color:var(--border-subtle)] px-5 py-4"
    >
        <h2 class="m-0 text-h3 font-semibold text-content-primary">
            {title}
        </h2>
        <button
            type="button"
            class="cursor-pointer rounded-[var(--radius-sm)] border-0 bg-transparent p-1.5 leading-none text-content-tertiary hover:bg-surface-sunken hover:text-content-primary focus-visible:outline-none focus-visible:shadow-[var(--focus-ring)] [&_ui5-icon]:size-icon-md"
            aria-label="Close details"
            onclick={requestClose}
            data-testid="side-panel-close"
        >
            <ui5-icon class="text-current" name="decline" aria-hidden="true"></ui5-icon>
        </button>
    </header>
    <div class="flex-1 overflow-auto px-5 py-4">
        {@render children()}
    </div>
    {#if footer}
        <footer
            class="border-t border-outline-subtle bg-surface-subtle px-5 py-3"
        >
            {@render footer()}
        </footer>
    {/if}
</dialog>

<style>
    .side-panel {
        display: flex;
        translate: 0 0;
        opacity: 1;
        transition:
            translate var(--motion-base, 240ms) cubic-bezier(var(--ease-out, 0.16, 1, 0.3, 1)),
            opacity var(--motion-base, 240ms) cubic-bezier(var(--ease-out, 0.16, 1, 0.3, 1)),
            display var(--motion-base, 240ms) allow-discrete,
            overlay var(--motion-base, 240ms) allow-discrete;
    }

    .side-panel:not([open]) {
        display: none;
        translate: 100% 0;
        opacity: 0;
    }

    @starting-style {
        .side-panel[open] {
            translate: 100% 0;
            opacity: 0;
        }
    }

    .side-panel::backdrop {
        opacity: 1;
        transition:
            opacity var(--motion-base, 240ms) cubic-bezier(var(--ease, 0.4, 0, 0.2, 1)),
            display var(--motion-base, 240ms) allow-discrete,
            overlay var(--motion-base, 240ms) allow-discrete;
    }

    .side-panel:not([open])::backdrop {
        opacity: 0;
    }

    @starting-style {
        .side-panel[open]::backdrop {
            opacity: 0;
        }
    }

    @media (prefers-reduced-motion: reduce) {
        .side-panel,
        .side-panel::backdrop {
            transition: none;
        }

        .side-panel:not([open]) {
            translate: 0 0;
            opacity: 1;
        }
    }

    @media (max-width: 767px) {
        .side-panel {
            width: 100vw !important;
        }

        .side-panel__resizer {
            display: none;
        }
    }
</style>

