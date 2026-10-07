<script lang="ts">
    /**
     * Stage-phase progress strip. Renders a horizontal row of thin,
     * pill-shaped bars, one per phase, each in one of four states:
     *
     *   • done    → 100 % green gradient fill
     *   • active  → 100 % blue  gradient fill (running)
     *   • failed  → 100 % red   gradient fill
     *   • pending →   0 % fill (empty track)
     *
     * Colour alone distinguishes `active` (running) from `done`
     * (completed); the bar width is intentionally uniform across all
     * three filled states so the strip reads as a coloured status
     * ribbon rather than a per-segment progress meter. This is a
     * deliberate divergence from the design-system reference (which
     * paints `active` at 60 %); label typography (semibold blue vs
     * medium green) still separates running from completed.
     *
     * The parent decides what the phases are and what state each is
     * in — this component is a passive dispatcher, styling only.
     */

    import type { StagePhaseItem, StagePhaseSize } from "./types.js";

    interface Props {
        /**
         * Ordered list of phases to render. Rendering is left-to-right;
         * every segment has equal width (`flex: 1 1 0`).
         */
        phases: StagePhaseItem[];
        /**
         * Density variant.
         *   • `compact` – 5 px bar, no labels (in-card mini strip)
         *   • `default` – 7 px bar + 12 px labels
         *   • `lg`      – 10 px bar + 13 px labels
         */
        size?: StagePhaseSize;
        /**
         * Optional aria-label for the whole strip. When provided, the
         * container is exposed as a `role="group"` region so assistive
         * tech announces the aggregate state (e.g. "Deploying vector").
         */
        ariaLabel?: string;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let {
        phases,
        size = "default",
        ariaLabel,
        class: className,
    }: Props = $props();

    const FILL_CLASSES = {
        active: "w-full bg-[image:var(--progress-active)]",
        done: "w-full bg-[image:var(--progress-done)]",
        failed: "w-full bg-[image:var(--progress-failed)]",
        pending: "w-0",
    } as const;

    const LABEL_CLASSES = {
        active: "text-status-deploying-fg font-semibold",
        done: "text-status-healthy-fg font-[var(--weight-medium)]",
        failed: "text-status-error-fg font-semibold",
        pending: "text-content-tertiary",
    } as const;
</script>

<div
    class={[
        "stage-progress flex flex-nowrap items-start",
        size === "compact" ? "stage-progress--compact gap-1" : "gap-[5px]",
        size === "lg" && "stage-progress--lg",
        className,
    ]}
    role={ariaLabel ? "group" : undefined}
    aria-label={ariaLabel}
>
    {#each phases as phase, index (index)}
        <div
            class={["stage-progress__seg min-w-0 flex-[1_1_0]", `stage-progress__seg--${phase.state}`]}
            data-state={phase.state}
            aria-current={phase.state === "active" ? "step" : undefined}
        >
            <div class={[
                "overflow-hidden rounded-full bg-[var(--track)]",
                size === "compact" ? "h-[5px]" : size === "lg" ? "h-2.5" : "h-[7px]",
            ]}><span class={["block h-full rounded-full", FILL_CLASSES[phase.state]]}></span></div>
            {#if size !== "compact"}
                <div class={[
                    "stage-progress__label flex items-center gap-1 overflow-hidden text-ellipsis whitespace-nowrap",
                    size === "lg" ? "mt-2 text-compact" : "mt-1.5 text-meta",
                    LABEL_CLASSES[phase.state],
                ]}>{phase.label}</div>
            {/if}
        </div>
    {/each}
</div>
