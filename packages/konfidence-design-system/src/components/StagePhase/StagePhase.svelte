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

    const containerClass = $derived.by(() => {
        const parts = ["stage-progress"];
        if (size === "compact") {
            parts.push("stage-progress--compact");
        } else if (size === "lg") {
            parts.push("stage-progress--lg");
        }
        if (className) {
            parts.push(className);
        }
        return parts.join(" ");
    });
</script>

<div
    class={containerClass}
    role={ariaLabel ? "group" : undefined}
    aria-label={ariaLabel}
>
    {#each phases as phase, index (index)}
        <div
            class={`stage-progress__seg stage-progress__seg--${phase.state}`}
            data-state={phase.state}
            aria-current={phase.state === "active" ? "step" : undefined}
        >
            <div class="stage-progress__bar"><span></span></div>
            {#if size !== "compact"}
                <div class="stage-progress__label">{phase.label}</div>
            {/if}
        </div>
    {/each}
</div>

<style>
    .stage-progress {
        display: flex;
        flex-direction: row;
        flex-wrap: nowrap;
        align-items: flex-start;
        gap: 5px;
    }
    .stage-progress__seg {
        flex: 1 1 0;
        min-width: 0;
    }
    .stage-progress__bar {
        height: 7px;
        border-radius: var(--radius-pill);
        background: var(--track);
        overflow: hidden;
    }
    .stage-progress__bar span {
        display: block;
        height: 100%;
        width: 0;
        border-radius: var(--radius-pill);
    }
    .stage-progress__seg--done .stage-progress__bar span {
        width: 100%;
        background: var(--progress-done);
    }
    .stage-progress__seg--active .stage-progress__bar span {
        width: 100%;
        background: var(--progress-active);
    }
    .stage-progress__seg--failed .stage-progress__bar span {
        width: 100%;
        background: var(--progress-failed);
    }
    .stage-progress__seg--pending .stage-progress__bar span {
        width: 0;
    }

    .stage-progress__label {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: var(--text-meta);
        color: var(--text-tertiary);
        margin-top: 6px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    .stage-progress__seg--done .stage-progress__label {
        color: var(--status-healthy-fg);
        font-weight: var(--weight-medium);
    }
    .stage-progress__seg--active .stage-progress__label {
        color: var(--status-deploying-fg);
        font-weight: var(--weight-semibold);
    }
    .stage-progress__seg--failed .stage-progress__label {
        color: var(--status-error-fg);
        font-weight: var(--weight-semibold);
    }

    /* Density variants. */
    .stage-progress--compact {
        gap: 4px;
    }
    .stage-progress--compact .stage-progress__bar {
        height: 5px;
    }
    .stage-progress--lg .stage-progress__bar {
        height: 10px;
    }
    .stage-progress--lg .stage-progress__label {
        font-size: var(--text-sm);
        margin-top: 8px;
    }
</style>
