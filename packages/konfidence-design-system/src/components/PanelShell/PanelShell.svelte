<script lang="ts">
    import { Dialog, Portal } from "@skeletonlabs/skeleton-svelte";
    import type { Snippet } from "svelte";
    import { onDestroy, tick, untrack } from "svelte";
    import "@ui5/webcomponents-icons/dist/decline.js";
    import "@ui5/webcomponents/dist/Icon.js";

    interface Props {
        open?: boolean;
        title: string;
        placement: "right" | "bottom";
        dialogClass?: string;
        dialogStyle?: string;
        backdropClass?: string;
        testid: string;
        closeLabel?: string;
        children: Snippet;
        footer?: Snippet;
        resizer?: Snippet;
        headerAction?: Snippet;
        onClose?: () => void;
        modal?: boolean;
        /** Changing this value refocuses the panel without replaying its animation. */
        focusKey?: string;
        /** Called after the closing transition has completed. */
        onCloseComplete?: () => void;
        /** Bindable reference to the panel surface for measurement. */
        panel?: HTMLDivElement;
    }

    let {
        open = $bindable(false),
        title,
        placement,
        dialogClass = "",
        dialogStyle = "",
        backdropClass = "",
        testid,
        closeLabel = "Close details",
        children,
        footer,
        resizer,
        headerAction,
        onClose,
        modal = true,
        focusKey,
        onCloseComplete,
        panel = $bindable(),
    }: Props = $props();

    let anchor: HTMLSpanElement | undefined = $state();
    let closeButton: HTMLButtonElement | undefined = $state();
    let dialogOpen = $state(false);
    let converting = $state(false);
    let conversionFocus: HTMLElement | undefined = undefined;
    let previousModal: boolean | undefined = undefined;
    let previouslyPresentedOpen = false;
    let previouslyOpen = false;
    let previouslyFocusedOpen = false;
    let previousFocusKey: string | undefined = undefined;
    let closing = false;
    let closeCompletionTimer: ReturnType<typeof setTimeout> | undefined = undefined;

    // The portal escapes clipping/stacking ancestors. Carry the local app-shell
    // offset with it instead of relying on inheritance from the original parent.
    const headerHeight = $derived(anchor
        ? getComputedStyle(anchor).height
        : "0px");

    const CLOSE_TRANSITION_BUFFER_MS = 50;
    const MILLISECONDS_PER_SECOND = 1000;

    const cssTimeToMilliseconds = (value: string): number => {
        const parsed = Number.parseFloat(value);
        return value.trim().endsWith("ms") ? parsed : parsed * MILLISECONDS_PER_SECOND;
    };

    const clearCloseTimer = (): void => {
        if (closeCompletionTimer !== undefined) {
            clearTimeout(closeCompletionTimer);
            closeCompletionTimer = undefined;
        }
    };

    const completeClose = (): void => {
        if (!closing || open) {
            return;
        }
        clearCloseTimer();
        closing = false;
        onCloseComplete?.();
    };

    const scheduleCloseCompletion = (): void => {
        // oxlint-disable-next-line eslint/max-statements -- CSS duration/delay lists determine the fallback deadline.
        queueMicrotask(() => {
            if (!closing || open) {
                return;
            }
            if (!panel) {
                completeClose();
                return;
            }
            const style = getComputedStyle(panel);
            const durations = style.transitionDuration.split(",").map(cssTimeToMilliseconds);
            const delays = style.transitionDelay.split(",").map(cssTimeToMilliseconds);
            const transitionTime = Math.max(0, ...durations.map((duration, index) =>
                duration + (delays[index] ?? delays[0] ?? 0)));
            if (transitionTime === 0) {
                completeClose();
                return;
            }
            closeCompletionTimer = setTimeout(completeClose, transitionTime + CLOSE_TRANSITION_BUFFER_MS);
        });
    };

    $effect(() => {
        if (open) {
            clearCloseTimer();
            closing = false;
        } else if (previouslyOpen) {
            closing = true;
            scheduleCloseCompletion();
        }
        previouslyOpen = open;
    });

    const restartDialog = (): (() => void) => {
        // Skeleton 5 starts modality effects on entry to its open state.
        // Cycle that state, not the DOM: keep content mounted and visually
        // open so changing modality preserves form state and the animation.
        conversionFocus = panel?.contains(document.activeElement)
            && document.activeElement instanceof HTMLElement
            ? document.activeElement : closeButton;
        converting = true;
        dialogOpen = false;
        let cancelled = false;
        void tick().then(() => {
            if (!cancelled) {
                dialogOpen = open;
                converting = false;
            }
        });
        return () => { cancelled = true; };
    };

    $effect(() => {
        const restart = open && previouslyPresentedOpen && previousModal !== modal;
        previouslyPresentedOpen = open;
        previousModal = modal;
        if (restart) {
            return untrack(restartDialog);
        }
        dialogOpen = open;
        converting = false;
        if (!open) {
            conversionFocus = undefined;
        }
    });

    $effect(() => {
        if (open && previouslyFocusedOpen && previousFocusKey !== focusKey) {
            queueMicrotask(() => closeButton?.focus({ preventScroll: true }));
        }
        previouslyFocusedOpen = open;
        previousFocusKey = focusKey;
    });

    // oxlint-disable-next-line unicorn/no-null -- Skeleton's MaybeElement requires null rather than undefined.
    const initialFocus = (): HTMLElement | null => conversionFocus ?? closeButton ?? null;

    const handleOpenChange = (details: { open: boolean }): void => {
        if (!details.open && open) {
            open = false;
            onClose?.();
        }
    };

    onDestroy(() => {
        closing = false;
        clearCloseTimer();
    });
</script>

<span bind:this={anchor} hidden style:height="var(--app-shell-header-height, 0px)"></span>

<Dialog
    open={dialogOpen}
    {modal}
    trapFocus={modal}
    preventScroll={modal}
    closeOnInteractOutside={modal}
    restoreFocus={!converting}
    initialFocusEl={initialFocus}
    onOpenChange={handleOpenChange}
    onRequestDismiss={(event) => {
        // Panels are independently controlled siblings, not nested dialogs.
        // Closing/reconfiguring an older panel must not dismiss newer ones.
        event.preventDefault();
    }}
>
    <Portal>
        <div class="contents" style:--app-shell-header-height={headerHeight}>
            <Dialog.Backdrop
                class="panel-shell-backdrop pointer-events-auto fixed inset-0 z-40 {backdropClass}"
                hidden={!open || !modal}
                data-state={open && modal ? "open" : "closed"}
                data-testid="{testid}-backdrop"
            />
            <Dialog.Positioner class="panel-shell-positioner pointer-events-none fixed inset-0 z-40">
                <Dialog.Content
                    class={[
                        "panel-shell pointer-events-auto fixed m-0 flex flex-col overflow-hidden border-0 bg-surface-default p-0 text-content-primary outline-none",
                        placement === "right" && "top-[var(--app-shell-header-height,0px)] right-0 bottom-0 left-auto h-auto max-h-[calc(100dvh-var(--app-shell-header-height,0px))] max-w-screen",
                        placement === "bottom" && "inset-x-0 bottom-0 top-auto max-h-[calc(100dvh-var(--app-shell-header-height,0px))] w-screen max-w-screen",
                        dialogClass,
                    ]}
                    style={dialogStyle}
                    hidden={!open}
                    data-state={open ? "open" : "closed"}
                    data-testid={testid}
                    data-placement={placement}
                    data-panel-surface
                    ontransitionend={(event) => {
                        if (event.target === panel && event.propertyName === "translate") {
                            completeClose();
                        }
                    }}
                >
                    {#snippet element(attributes)}
                        <div {...attributes} bind:this={panel}>
                            {@render resizer?.()}
                            <header class="flex items-center justify-between gap-2 border-b border-[color:var(--border-subtle)] px-5 py-4">
                                <Dialog.Title>
                                    {#snippet element(titleAttributes)}
                                        <h2 {...titleAttributes} class="m-0 text-h3 font-semibold text-content-primary">{title}</h2>
                                    {/snippet}
                                </Dialog.Title>
                                <div class="flex items-center gap-1">
                                    {@render headerAction?.()}
                                    <Dialog.CloseTrigger
                                        class="cursor-pointer rounded-[var(--radius-sm)] border-0 bg-transparent p-1.5 leading-none text-content-tertiary hover:bg-surface-sunken hover:text-content-primary focus-visible:outline-none focus-visible:shadow-[var(--focus-ring)] [&_ui5-icon]:size-icon-md"
                                        aria-label={closeLabel}
                                        data-testid="{testid}-close"
                                    >
                                        {#snippet element(closeAttributes)}
                                            <button {...closeAttributes} bind:this={closeButton}>
                                                <ui5-icon class="text-current" name="decline" aria-hidden="true"></ui5-icon>
                                            </button>
                                        {/snippet}
                                    </Dialog.CloseTrigger>
                                </div>
                            </header>
                            <div class="flex-1 overflow-auto px-5 py-4">
                                {@render children()}
                            </div>
                            {#if footer}
                                <footer class="border-t border-outline-subtle bg-surface-subtle px-5 py-3">
                                    {@render footer()}
                                </footer>
                            {/if}
                        </div>
                    {/snippet}
                </Dialog.Content>
            </Dialog.Positioner>
        </div>
    </Portal>
</Dialog>

<style>
    :global(.panel-shell) {
        --panel-closed-translate: 0 100%;

        translate: 0 0;
        opacity: 1;
        transition:
            translate var(--motion-base, 240ms) cubic-bezier(var(--ease-out, 0.16, 1, 0.3, 1)),
            opacity var(--motion-base, 240ms) cubic-bezier(var(--ease-out, 0.16, 1, 0.3, 1)),
            display var(--motion-base, 240ms) allow-discrete;
    }

    :global(.panel-shell[data-placement="right"]) {
        --panel-closed-translate: 100% 0;
    }

    :global(.panel-shell[data-state="closed"]) {
        display: none;
        translate: var(--panel-closed-translate);
        opacity: 0;
    }

    @starting-style {
        :global(.panel-shell[data-state="open"]) {
            translate: var(--panel-closed-translate);
            opacity: 0;
        }
    }

    :global(.panel-shell-backdrop) {
        z-index: calc(40 + var(--layer-index, 0) * 2);
        background: var(--scrim);
        opacity: 1;
        transition:
            opacity var(--motion-base, 240ms) cubic-bezier(var(--ease, 0.4, 0, 0.2, 1)),
            display var(--motion-base, 240ms) allow-discrete;
    }

    :global(.panel-shell-positioner) {
        z-index: calc(41 + var(--layer-index, 0) * 2);
    }

    :global(.panel-shell-backdrop[data-state="closed"]) {
        display: none;
        opacity: 0;
    }

    @starting-style {
        :global(.panel-shell-backdrop[data-state="open"]) {
            opacity: 0;
        }
    }

    @media (prefers-reduced-motion: reduce) {
        :global(.panel-shell),
        :global(.panel-shell-backdrop) {
            transition: none;
        }
    }
</style>
