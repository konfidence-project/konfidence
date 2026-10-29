<script lang="ts">
    import { clampPanelSize } from "./panel-size";

    interface Props {
        edge: "left" | "top";
        size: number;
        min: number;
        max: number;
        testid: string;
        /** Reports the committed size after a drag or keyboard adjustment. */
        onResize?: (size: number) => void;
    }

    const KEYBOARD_STEP = 16;

    let { edge, size = $bindable(), min, max, testid, onResize }: Props = $props();

    const effectiveSize = $derived(clampPanelSize(size, min, max));
    let pointerId: number | undefined = undefined;
    let dragStart = 0;
    let dragStartSize = 0;

    const coordinate = (event: PointerEvent): number =>
        edge === "left" ? event.clientX : event.clientY;

    const commitSize = (next: number): void => {
        size = clampPanelSize(next, min, max);
        onResize?.(size);
    };

    const handlePointerDown = (event: PointerEvent): void => {
        if (event.button !== 0 || pointerId !== undefined) {
            return;
        }
        event.preventDefault();
        ({ pointerId } = event);
        dragStart = coordinate(event);
        dragStartSize = effectiveSize;
        (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    };

    const handlePointerMove = (event: PointerEvent): void => {
        if (pointerId !== event.pointerId) {
            return;
        }
        // Moving the left/top edge towards smaller coordinates grows the panel.
        size = clampPanelSize(dragStartSize + dragStart - coordinate(event), min, max);
    };

    const finishDrag = (event: PointerEvent): void => {
        if (pointerId !== event.pointerId) {
            return;
        }
        pointerId = undefined;
        const target = event.currentTarget as HTMLElement;
        if (target.hasPointerCapture(event.pointerId)) {
            target.releasePointerCapture(event.pointerId);
        }
        onResize?.(effectiveSize);
    };

    const handleKeydown = (event: KeyboardEvent): void => {
        const growKey = edge === "left" ? "ArrowLeft" : "ArrowUp";
        const shrinkKey = edge === "left" ? "ArrowRight" : "ArrowDown";
        const next = {
            End: min,
            Home: max,
            [growKey]: effectiveSize + KEYBOARD_STEP,
            [shrinkKey]: effectiveSize - KEYBOARD_STEP,
        }[event.key];
        if (next === undefined) {
            return;
        }
        event.preventDefault();
        commitSize(next);
    };
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
    class={[
        "panel-resizer absolute z-10 touch-none bg-transparent transition-colors hover:bg-[var(--border-strong,rgba(120,90,40,0.5))] focus-visible:bg-[var(--border-strong,rgba(120,90,40,0.7))] focus-visible:outline-none",
        edge === "left" ? "inset-y-0 left-0 w-1.5 cursor-ew-resize" : "inset-x-0 top-0 h-1.5 cursor-ns-resize",
    ]}
    role="separator"
    aria-orientation={edge === "left" ? "vertical" : "horizontal"}
    aria-label="Resize panel"
    aria-valuemin={min}
    aria-valuemax={max}
    aria-valuenow={effectiveSize}
    tabindex="0"
    data-testid={testid}
    onpointerdown={handlePointerDown}
    onpointermove={handlePointerMove}
    onpointerup={finishDrag}
    onpointercancel={finishDrag}
    onlostpointercapture={finishDrag}
    onkeydown={handleKeydown}
></div>

<style>
    @media (max-width: 767px) {
        .panel-resizer {
            display: none;
        }
    }
</style>
