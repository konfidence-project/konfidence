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
    }

    let { open = $bindable(false), title, children, footer, onClose }: Props = $props();

    let dialog: HTMLDialogElement | undefined = $state();

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
</script>

<dialog
    bind:this={dialog}
    class="fixed inset-y-0 right-0 left-auto m-0 h-[100dvh] max-h-[100dvh] w-[min(28rem,100vw)] max-w-[100vw] flex-col overflow-hidden border-0 border-l border-l-outline-subtle bg-surface-default p-0 text-content-primary shadow-[-8px_0_24px_rgba(0,0,0,0.12)] backdrop:bg-[rgba(20,17,11,0.32)] [&:not([open])]:hidden [&[open]]:flex"
    aria-label={title}
    data-testid="side-panel"
    onclose={requestClose}
    onclick={handleBackdropClick}
>
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
            <ui5-icon name="decline" aria-hidden="true"></ui5-icon>
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
