<script lang="ts">
    import type { Snippet } from "svelte";
    import "@ui5/webcomponents-icons/dist/decline.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import {
        isTopmostNonModalPanel,
        registerNonModalPanel,
        unregisterNonModalPanel,
    } from "./panel-stack";

    type Placement = "right" | "bottom";

    interface Props {
        open?: boolean;
        title: string;
        placement: Placement;
        /** Extra class applied to the dialog, e.g. placement-specific geometry/backdrop. */
        dialogClass?: string;
        /** Inline style applied to the dialog, e.g. the resolved width/height. */
        dialogStyle?: string;
        testid: string;
        closeLabel?: string;
        children: Snippet;
        footer?: Snippet;
        /** Resize affordance (resizer handle) rendered at the panel edge. */
        resizer?: Snippet;
        /** Optional extra header action rendered left of the close button (e.g. open full page). */
        headerAction?: Snippet;
        onClose?: () => void;
        /** Whether the dialog makes the surrounding page inert. */
        modal?: boolean;
        /** Changing this value moves focus to the close button without replaying the panel animation. */
        focusKey?: string;
        /** Called after the closing transition has completed. */
        onCloseComplete?: () => void;
        /** Bindable reference to the underlying dialog element for measurement. */
        dialog?: HTMLDialogElement;
    }

    let {
        open = $bindable(false),
        title,
        placement,
        dialogClass = "",
        dialogStyle = "",
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
        dialog = $bindable(),
    }: Props = $props();

    let closeButton: HTMLButtonElement | undefined = $state();
    let presentedModally: boolean | undefined = undefined;
    let previouslyOpen = false;
    let previousFocusKey: string | undefined = undefined;
    let closing = false;
    let closeCompletionTimer: ReturnType<typeof setTimeout> | undefined = undefined;
    const panelInstance = Symbol("panel-shell");

    const CLOSE_TRANSITION_BUFFER_MS = 50;
    const MILLISECONDS_PER_SECOND = 1000;

    const cssTimeToMilliseconds = (value: string): number => {
        const parsed = Number.parseFloat(value);
        return value.trim().endsWith("ms") ? parsed : parsed * MILLISECONDS_PER_SECOND;
    };

    const showDialog = (target: HTMLDialogElement): void => {
        if (modal) {
            target.showModal();
        } else {
            target.show();
        }
        presentedModally = modal;
    };

    const convertDialogModality = (target: HTMLDialogElement): void => {
        const focusedElement = target.contains(document.activeElement)
            ? (document.activeElement as HTMLElement)
            : undefined;
        target.close();
        showDialog(target);
        queueMicrotask(() =>
            (focusedElement ?? closeButton)?.focus({ preventScroll: true }),
        );
    };

    const completeClose = (): void => {
        if (!closing) {
            return;
        }
        if (closeCompletionTimer !== undefined) {
            clearTimeout(closeCompletionTimer);
            closeCompletionTimer = undefined;
        }
        closing = false;
        onCloseComplete?.();
    };

    $effect(() => {
        const target = dialog;
        if (!target) {
            return;
        }
        if (open && !target.open) {
            showDialog(target);
        } else if (open && presentedModally !== modal) {
            // Native dialogs cannot change modality while open. Closing and
            // reopening in the same task avoids replaying the CSS transition.
            convertDialogModality(target);
        } else if (!open && target.open) {
            closing = true;
            target.close();
        }
    });

    $effect(() => {
        const shouldFocus = open && (!previouslyOpen || focusKey !== previousFocusKey);
        previouslyOpen = open;
        previousFocusKey = focusKey;
        if (!shouldFocus) {
            return;
        }
        queueMicrotask(() => closeButton?.focus({ preventScroll: true }));
    });

    $effect(() => {
        if (!open || modal) {
            return;
        }
        registerNonModalPanel(panelInstance);
        // oxlint-disable-next-line unicorn/consistent-function-scoping -- closes over this panel's requestClose.
        const handleEscape = (event: KeyboardEvent): void => {
            if (
                event.key !== "Escape" ||
                !isTopmostNonModalPanel(panelInstance)
            ) {
                return;
            }
            event.preventDefault();
            event.stopImmediatePropagation();
            requestClose();
        };
        globalThis.document.addEventListener("keydown", handleEscape, true);
        return () => {
            globalThis.document.removeEventListener("keydown", handleEscape, true);
            unregisterNonModalPanel(panelInstance);
        };
    });

    const requestClose = (): void => {
        if (!open) {
            return;
        }
        open = false;
        onClose?.();
    };

    const handleBackdropClick = (event: MouseEvent): void => {
        if (modal && event.target === dialog) {
            requestClose();
        }
    };

    const scheduleCloseCompletion = (): void => {
        // oxlint-disable-next-line eslint/max-statements -- transition timing requires parsing duration and delay lists together.
        queueMicrotask(() => {
            if (!dialog) {
                completeClose();
                return;
            }
            const style = getComputedStyle(dialog);
            const durations = style.transitionDuration
                .split(",")
                .map(cssTimeToMilliseconds);
            const delays = style.transitionDelay
                .split(",")
                .map(cssTimeToMilliseconds);
            const transitionTime = Math.max(
                0,
                ...durations.map((duration, index) => duration + (delays[index] ?? delays[0] ?? 0)),
            );
            if (transitionTime === 0) {
                completeClose();
                return;
            }
            closeCompletionTimer = setTimeout(
                completeClose,
                transitionTime + CLOSE_TRANSITION_BUFFER_MS,
            );
        });
    };

    const handleNativeClose = (): void => {
        // A live modality conversion closes and immediately reopens the same
        // element; it is not a user dismissal.
        if (dialog?.open) {
            return;
        }
        closing = true;
        requestClose();
        scheduleCloseCompletion();
    };
</script>

<dialog
    bind:this={dialog}
    class={[
        "panel-shell fixed z-40 m-0 flex flex-col overflow-hidden border-0 bg-surface-default p-0 text-content-primary",
        placement === "right" &&
            "top-[var(--app-shell-header-height,0px)] right-0 bottom-0 left-auto h-auto max-h-[calc(100dvh-var(--app-shell-header-height,0px))] max-w-screen",
        placement === "bottom" &&
            "inset-x-0 bottom-0 top-auto max-h-[calc(100dvh-var(--app-shell-header-height,0px))] w-screen max-w-screen",
        dialogClass,
    ]}
    style={dialogStyle}
    aria-label={title}
    data-testid={testid}
    data-placement={placement}
    data-panel-surface
    onclose={handleNativeClose}
    onclick={handleBackdropClick}
    ontransitionend={(event) => {
        if (event.target === dialog && event.propertyName === "translate") {
            completeClose();
        }
    }}
>
    {@render resizer?.()}
    <header
        class="flex items-center justify-between gap-2 border-b border-[color:var(--border-subtle)] px-5 py-4"
    >
        <h2 class="m-0 text-h3 font-semibold text-content-primary">
            {title}
        </h2>
        <div class="flex items-center gap-1">
            {@render headerAction?.()}
            <button
                bind:this={closeButton}
                type="button"
                class="cursor-pointer rounded-[var(--radius-sm)] border-0 bg-transparent p-1.5 leading-none text-content-tertiary hover:bg-surface-sunken hover:text-content-primary focus-visible:outline-none focus-visible:shadow-[var(--focus-ring)] [&_ui5-icon]:size-icon-md"
                aria-label={closeLabel}
                onclick={requestClose}
                data-testid="{testid}-close"
            >
                <ui5-icon class="text-current" name="decline" aria-hidden="true"></ui5-icon>
            </button>
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
</dialog>

<style>
    .panel-shell {
        --panel-closed-translate: 0 100%;

        translate: 0 0;
        opacity: 1;
        transition:
            translate var(--motion-base, 240ms) cubic-bezier(var(--ease-out, 0.16, 1, 0.3, 1)),
            opacity var(--motion-base, 240ms) cubic-bezier(var(--ease-out, 0.16, 1, 0.3, 1)),
            display var(--motion-base, 240ms) allow-discrete,
            overlay var(--motion-base, 240ms) allow-discrete;
    }

    .panel-shell[data-placement="right"] {
        --panel-closed-translate: 100% 0;
    }

    .panel-shell:not([open]) {
        display: none;
        translate: var(--panel-closed-translate);
        opacity: 0;
    }

    @starting-style {
        .panel-shell[open] {
            translate: var(--panel-closed-translate);
            opacity: 0;
        }
    }

    .panel-shell::backdrop {
        opacity: 1;
        transition:
            opacity var(--motion-base, 240ms) cubic-bezier(var(--ease, 0.4, 0, 0.2, 1)),
            display var(--motion-base, 240ms) allow-discrete,
            overlay var(--motion-base, 240ms) allow-discrete;
    }

    .panel-shell:not([open])::backdrop {
        opacity: 0;
    }

    @starting-style {
        .panel-shell[open]::backdrop {
            opacity: 0;
        }
    }

    @media (prefers-reduced-motion: reduce) {
        .panel-shell,
        .panel-shell::backdrop {
            transition: none;
        }

        .panel-shell:not([open]) {
            translate: 0 0;
            opacity: 1;
        }
    }
</style>
