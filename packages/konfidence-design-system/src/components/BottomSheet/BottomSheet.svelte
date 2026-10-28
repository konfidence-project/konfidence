<script lang="ts">
    import type { Snippet } from "svelte";
    import PanelShell from "../PanelShell/PanelShell.svelte";
    import PanelResizer from "../PanelResizer/PanelResizer.svelte";
    import { clampPanelSize } from "../PanelResizer/panel-size";

    interface Props {
        open?: boolean;
        title: string;
        children: Snippet;
        footer?: Snippet;
        onClose?: () => void;
        height?: number;
        onHeightChange?: (height: number) => void;
        modal?: boolean;
        focusKey?: string;
        onCloseComplete?: () => void;
    }

    const DEFAULT_HEIGHT = 420;
    // The minimum keeps the header and footer visible so the sheet can never
    // collapse to a sliver; the maximum is computed from the viewport minus the
    // app-shell header so the sheet never exceeds the viewport.
    const MIN_HEIGHT = 220;
    const FALLBACK_VIEWPORT_HEIGHT = 800;

    let {
        open = $bindable(false),
        title,
        children,
        footer,
        onClose,
        height = $bindable(DEFAULT_HEIGHT),
        onHeightChange,
        modal = true,
        focusKey,
        onCloseComplete,
    }: Props = $props();

    let dialog: HTMLDialogElement | undefined = $state();
    let viewportHeight = $state(
        typeof globalThis.innerHeight === "number"
            ? globalThis.innerHeight
            : FALLBACK_VIEWPORT_HEIGHT,
    );

    const handleResize = (): void => {
        viewportHeight = globalThis.innerHeight;
    };

    $effect(() => {
        globalThis.addEventListener("resize", handleResize);
        handleResize();
        return () => globalThis.removeEventListener("resize", handleResize);
    });

    const headerOffset = $derived.by((): number => {
        const target = dialog;
        if (target === undefined) {
            return 0;
        }
        const raw = getComputedStyle(target).getPropertyValue("--app-shell-header-height");
        const parsed = Number.parseFloat(raw);
        return Number.isFinite(parsed) ? parsed : 0;
    });

    const maxHeight = $derived(Math.max(MIN_HEIGHT, Math.round(viewportHeight - headerOffset)));
    const effectiveHeight = $derived(clampPanelSize(height, MIN_HEIGHT, maxHeight));
</script>

<PanelShell
    bind:open
    bind:dialog
    {title}
    placement="bottom"
    testid="bottom-sheet"
    dialogClass="bottom-sheet border-t border-t-outline-subtle shadow-[0_-8px_24px_rgba(0,0,0,0.12)]"
    dialogStyle="height: {effectiveHeight}px"
    {onClose}
    {modal}
    {focusKey}
    {onCloseComplete}
    {footer}
>
    {#snippet resizer()}
        <PanelResizer
            edge="top"
            bind:size={height}
            min={MIN_HEIGHT}
            max={maxHeight}
            testid="bottom-sheet-resizer"
            onResize={onHeightChange}
        />
    {/snippet}
    {@render children()}
</PanelShell>

<style>
    /* Lighter, theme-aware backdrop scoped to BottomSheet so the landscape
       behind the sheet stays legible. The global --scrim token and the right
       panel's backdrop are left unchanged. */
    :global(.bottom-sheet) {
        --bottom-sheet-scrim: rgba(45, 30, 8, 0.1);
    }

    :global(.dark .bottom-sheet) {
        --bottom-sheet-scrim: rgba(0, 0, 0, 0.22);
    }

    :global(.bottom-sheet::backdrop) {
        background: var(--bottom-sheet-scrim);
    }

</style>
