<script lang="ts">
    import type { StageHealth } from "./utils/stage-view.js";

    type Tone = "Info" | StageHealth;

    const {
        status,
        label = status,
        pulse = false,
    } = $props<{
        status: Tone;
        label?: string;
        pulse?: boolean;
    }>();
</script>

<span class={["status-pill", `status-${status.toLowerCase()}`]}>
    <span class="dot" class:pulse></span>
    <span class="label">{label}</span>
</span>

<style>
    .status-pill {
        display: inline-flex;
        align-items: center;
        gap: 0.3125rem;
        width: fit-content;
        padding: 0.1875rem 0.5625rem;
        border-radius: 999px;
        font-size: 0.6875rem;
        font-weight: 700;
        line-height: 1;
        letter-spacing: 0.03em;
        text-transform: uppercase;
        white-space: nowrap;
        font-family:
            "SF Mono", "Menlo", "Consolas", ui-monospace, monospace;
    }

    .dot {
        width: 0.375rem;
        height: 0.375rem;
        border-radius: 50%;
        background: currentColor;
        flex-shrink: 0;
    }

    .dot.pulse {
        animation: pill-pulse 0.8s ease-in-out infinite alternate;
    }

    @keyframes pill-pulse {
        from {
            opacity: 1;
        }
        to {
            opacity: 0.25;
        }
    }

    /* Legacy tones (kept for backwards compatibility with existing callers). */
    .status-info {
        color: var(--sapInformativeTextColor);
        background: var(--sapInformationBackground);
    }
    .status-true {
        color: var(--sapPositiveTextColor);
        background: var(--sapSuccessBackground);
    }
    .status-false {
        color: var(--sapNegativeTextColor);
        background: var(--sapErrorBackground);
    }
    .status-unknown {
        color: var(--sapCriticalTextColor);
        background: var(--sapWarningBackground);
    }

    /* Mockup-parity tones. */
    .status-healthy {
        color: var(--sapPositiveTextColor, #2e7d32);
        background: var(--sapSuccessBackground, #e8f5e9);
    }
    .status-deploying {
        color: var(--sapInformativeTextColor, #1565c0);
        background: var(--sapInformationBackground, #e3f2fd);
    }
    .status-warning {
        color: var(--sapCriticalTextColor, #f57f17);
        background: var(--sapWarningBackground, #fff8e1);
    }
    .status-error {
        color: var(--sapNegativeTextColor, #c62828);
        background: var(--sapErrorBackground, #ffebee);
    }
</style>
